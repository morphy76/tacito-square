package observability

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/shared/tenant"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// NATSHeaderCarrier adapts a nats.Header to satisfy propagation.TextMapCarrier,
// enabling the global OTel propagator to Inject and Extract W3C trace context
// (traceparent / tracestate) directly on NATS messages.
type NATSHeaderCarrier natsclient.Header

// Get returns the first value associated with the given key, or "" if absent.
func (c NATSHeaderCarrier) Get(key string) string {
	return natsclient.Header(c).Get(key)
}

// Set stores the value under the given key, overwriting any previous value.
func (c NATSHeaderCarrier) Set(key, value string) {
	natsclient.Header(c).Set(key, value)
}

// Keys returns all header keys present in the carrier.
func (c NATSHeaderCarrier) Keys() []string {
	h := natsclient.Header(c)
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// InjectNATSContext serialises the active OTel span context from ctx into
// msg.Header using the global TextMapPropagator. It initialises msg.Header
// if nil so callers do not need to do so manually.
func InjectNATSContext(ctx context.Context, msg *natsclient.Msg) {
	if msg.Header == nil {
		msg.Header = make(natsclient.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, NATSHeaderCarrier(msg.Header))
}

// ExtractNATSContext recovers an OTel span context previously injected by
// InjectNATSContext and returns a new context that carries it. If msg.Header
// is nil or contains no trace information, ctx is returned unchanged.
func ExtractNATSContext(ctx context.Context, msg *natsclient.Msg) context.Context {
	if msg.Header == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, NATSHeaderCarrier(msg.Header))
}

// natsContextAttrs builds the standard event attribute set for a NATS span event:
// tenant_id (from context, if available) plus any caller-supplied extras.
func natsContextAttrs(ctx context.Context, extra ...attribute.KeyValue) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 1+len(extra))
	if t := tenant.FromContext(ctx); t != nil {
		attrs = append(attrs, attribute.String("tenant_id", t.FullName()))
	}
	return append(attrs, extra...)
}

// RequestMsgWithTrace is a drop-in replacement for nc.RequestMsgWithContext that:
//   - Initialises msg.Header and injects the W3C traceparent from ctx.
//   - Wraps the outbound call in a SpanKindClient span named "nats.request" with
//     messaging.system and messaging.destination attributes.
//   - Emits a "nats.send" event (with tenant_id if present) just before the call,
//     a "nats.reply.received" event on success, and a "nats.error" event on failure.
//   - Records the round-trip duration in OutboundDependencyDuration{dependency="nats"}.
//
// Future NATS publishers should use this function instead of calling
// nc.RequestMsgWithContext directly.
func RequestMsgWithTrace(ctx context.Context, nc *natsclient.Conn, subject string, msg *natsclient.Msg) (*natsclient.Msg, error) {
	tracer := otel.Tracer("nats")
	ctx, span := tracer.Start(ctx, "nats.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination", subject),
		),
	)
	defer span.End()

	// Inject the active span context so the consumer can link its work back to this trace.
	InjectNATSContext(ctx, msg)

	// "nats.send": timestamped annotation visible in Zipkin/Jaeger just before the wire call.
	span.AddEvent("nats.send", trace.WithAttributes(
		natsContextAttrs(ctx,
			attribute.String("messaging.destination", subject),
		)...,
	))

	start := time.Now()
	reply, err := nc.RequestMsgWithContext(ctx, msg)
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.AddEvent("nats.error", trace.WithAttributes(
			natsContextAttrs(ctx,
				attribute.String("messaging.destination", subject),
				attribute.String("error.message", err.Error()),
			)...,
		))
	} else {
		span.AddEvent("nats.reply.received", trace.WithAttributes(
			natsContextAttrs(ctx,
				attribute.Int("reply.size_bytes", len(reply.Data)),
			)...,
		))
	}

	OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "nats"),
			attribute.String("status", status),
		),
	)

	return reply, err
}

// NATSHandlerFunc is the inner handler signature used by WrapNATSHandler.
// It receives the propagated context (carrying the consumer span), a logger
// enriched with trace_id / span_id, and the original NATS message.
// Returning a non-nil error causes WrapNATSHandler to record an error event
// on the span, enabling uniform error observability across all NATS handlers.
type NATSHandlerFunc func(ctx context.Context, logger zerolog.Logger, msg *natsclient.Msg) error

// WrapNATSHandler converts a NATSHandlerFunc into a natsclient.MsgHandler that
// automatically:
//   - Extracts the OTel trace context from msg.Header.
//   - Starts a SpanKindConsumer span named spanName with messaging attributes.
//   - Enriches logger with trace_id and span_id via observability.WithContext.
//   - Emits a "nats.receive" event with subject, payload size, and tenant_id.
//   - If inner returns an error, records it on the span with a "nats.handler.error" event.
//
// Future NATS subscribers should wrap their handlers with this function so they
// participate in distributed traces without any additional OTel wiring.
func WrapNATSHandler(spanName string, logger zerolog.Logger, inner NATSHandlerFunc) natsclient.MsgHandler {
	return func(msg *natsclient.Msg) {
		ctx := ExtractNATSContext(context.Background(), msg)

		tracer := otel.Tracer("nats")
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("messaging.system", "nats"),
				attribute.String("messaging.destination", msg.Subject),
			),
		)
		defer span.End()

		// "nats.receive": timestamped annotation visible in Zipkin/Jaeger on arrival.
		span.AddEvent("nats.receive", trace.WithAttributes(
			natsContextAttrs(ctx,
				attribute.String("messaging.subject", msg.Subject),
				attribute.Int("payload.size_bytes", len(msg.Data)),
			)...,
		))

		enrichedLogger := WithContext(logger, ctx)
		if err := inner(ctx, enrichedLogger, msg); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.AddEvent("nats.handler.error", trace.WithAttributes(
				natsContextAttrs(ctx,
					attribute.String("messaging.subject", msg.Subject),
					attribute.String("error.message", err.Error()),
				)...,
			))
			enrichedLogger.Error().Err(err).
				Str("span_name", spanName).
				Str("subject", msg.Subject).
				Msg("NATS handler returned an error")
		}
	}
}

// RecordNATSError stamps a span with both a RecordError (exception semantics) and a
// "nats.handler.error" event (structured troubleshooting). Use this inside a
// NATSHandlerFunc for domain-level errors that are handled locally but should still
// appear in the trace.
func RecordNATSError(span trace.Span, ctx context.Context, subject string, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.AddEvent("nats.handler.error", trace.WithAttributes(
		natsContextAttrs(ctx,
			attribute.String("messaging.subject", subject),
			attribute.String("error.message", err.Error()),
		)...,
	))
}

// SpanFromNATSContext returns the active span from ctx. Use inside a NATSHandlerFunc
// to emit ad-hoc domain events without importing the trace package in the adapter layer.
func SpanFromNATSContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}
