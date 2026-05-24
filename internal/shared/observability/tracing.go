package observability

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer sets up OpenTelemetry tracing with OTLP gRPC exporter.
// Returns a shutdown function that must be called on application exit.
func InitTracer(ctx context.Context, serviceName, serviceVersion, otelEndpoint string) (func(context.Context) error, error) {
	if otelEndpoint == "" {
		// No-op: return a noop shutdown if OTel is not configured
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// TracingMiddleware returns a Gin middleware that extracts OTel trace context from incoming HTTP headers
// and sets it in the request context, starting a server span.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract incoming trace parent context using global TextMapPropagator
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start a new server span using extracted context
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.FullPath(), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		// Pass trace context down to Gin request context
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// Record span status based on status code
		status := c.Writer.Status()
		if status >= 500 {
			span.RecordError(fmt.Errorf("HTTP request failed with status: %d", status))
		}
	}
}
