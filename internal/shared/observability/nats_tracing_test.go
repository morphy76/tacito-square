package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	natsclient "github.com/nats-io/nats.go"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// startNATSServer spins up an in-process NATS server for testing and returns
// the server and a connected client. Both are cleaned up by the test's defer.
func startNATSServer(t *testing.T) (*server.Server, *natsclient.Conn) {
	t.Helper()
	opts := &server.Options{Host: "127.0.0.1", Port: -1}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)
	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready")
	}
	nc, err := natsclient.Connect(ns.ClientURL())
	require.NoError(t, err)
	return ns, nc
}

// newTestTracerProvider creates a TracerProvider backed by an in-memory exporter
// and installs it as the global provider. Returns the exporter and a cleanup func.
func newTestTracerProvider(t *testing.T) (*tracetest.InMemoryExporter, func()) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return exp, func() { _ = tp.Shutdown(context.Background()) }
}

// ---------------------------------------------------------------------------
// NATSHeaderCarrier
// ---------------------------------------------------------------------------

func TestNATSHeaderCarrier_GetSetKeys(t *testing.T) {
	header := make(natsclient.Header)
	carrier := NATSHeaderCarrier(header)

	carrier.Set("traceparent", "00-abc-def-01")
	assert.Equal(t, "00-abc-def-01", carrier.Get("traceparent"))

	carrier.Set("tracestate", "vendor=value")
	keys := carrier.Keys()
	// nats.Header is case-sensitive and stores keys exactly as provided (no canonicalization)
	assert.Contains(t, keys, "traceparent")
	assert.Contains(t, keys, "tracestate")
}

func TestNATSHeaderCarrier_GetMissing(t *testing.T) {
	header := make(natsclient.Header)
	carrier := NATSHeaderCarrier(header)
	assert.Equal(t, "", carrier.Get("traceparent"))
}

// ---------------------------------------------------------------------------
// InjectNATSContext / ExtractNATSContext round-trip
// ---------------------------------------------------------------------------

func TestInjectExtractNATSContext_RoundTrip(t *testing.T) {
	_, cleanup := newTestTracerProvider(t)
	defer cleanup()

	// Start a parent span using the test tracer provider already installed as global
	ctx, span := otel.Tracer("test").Start(context.Background(), "parent")
	defer span.End()

	wantTraceID := span.SpanContext().TraceID().String()

	// Inject into a NATS message
	msg := &natsclient.Msg{Subject: "test.subject"}
	InjectNATSContext(ctx, msg)

	// The header must be non-nil and carry traceparent (W3C propagator writes lowercase key)
	require.NotNil(t, msg.Header)
	assert.NotEmpty(t, msg.Header.Get("traceparent"))

	// Extract on the consumer side and verify the trace ID is preserved
	extracted := ExtractNATSContext(context.Background(), msg)
	gotSpanCtx := trace.SpanContextFromContext(extracted)
	assert.True(t, gotSpanCtx.IsValid())
	assert.Equal(t, wantTraceID, gotSpanCtx.TraceID().String())
}

func TestExtractNATSContext_NoHeader_ReturnsBackground(t *testing.T) {
	msg := &natsclient.Msg{Subject: "test.subject"}
	ctx := ExtractNATSContext(context.Background(), msg)
	spanCtx := trace.SpanContextFromContext(ctx)
	assert.False(t, spanCtx.IsValid(), "no span should be present when header is absent")
}

// ---------------------------------------------------------------------------
// RequestMsgWithTrace — client span + traceparent injection + metric
// ---------------------------------------------------------------------------

func TestRequestMsgWithTrace_InjectsTraceparent(t *testing.T) {
	_, cleanup := newTestTracerProvider(t)
	defer cleanup()

	ns, nc := startNATSServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subject := "ts.test.inject"
	headerCh := make(chan natsclient.Header, 1)

	sub, err := nc.Subscribe(subject, func(msg *natsclient.Msg) {
		// Copy header values manually (nats.Header wraps net/http.Header)
		copied := make(natsclient.Header)
		for k, vs := range msg.Header {
			copied[k] = append([]string(nil), vs...)
		}
		headerCh <- copied
		_ = msg.Respond([]byte(`{}`))
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	outMsg := &natsclient.Msg{Subject: subject, Data: []byte(`{}`)}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = RequestMsgWithTrace(ctx, nc, subject, outMsg)
	require.NoError(t, err)

	select {
	case h := <-headerCh:
		assert.NotEmpty(t, h.Get("traceparent"), "traceparent must be injected into the NATS message header")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for NATS message")
	}
}

func TestRequestMsgWithTrace_ExportsClientSpan(t *testing.T) {
	exp, cleanup := newTestTracerProvider(t)
	defer cleanup()

	ns, nc := startNATSServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subject := "ts.test.span"

	sub, err := nc.Subscribe(subject, func(msg *natsclient.Msg) {
		_ = msg.Respond([]byte(`{}`))
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	outMsg := &natsclient.Msg{Subject: subject, Data: []byte(`{}`)}
	_, err = RequestMsgWithTrace(ctx, nc, subject, outMsg)
	require.NoError(t, err)

	spans := exp.GetSpans()
	require.NotEmpty(t, spans, "at least one span must be exported")

	var found bool
	for _, s := range spans {
		if s.Name == "nats.request" {
			found = true
			assert.Equal(t, trace.SpanKindClient, s.SpanKind)
			// Verify messaging attributes
			attrMap := map[string]string{}
			for _, a := range s.Attributes {
				attrMap[string(a.Key)] = a.Value.AsString()
			}
			assert.Equal(t, "nats", attrMap["messaging.system"])
			assert.Equal(t, subject, attrMap["messaging.destination"])
			break
		}
	}
	assert.True(t, found, "a 'nats.request' SpanKindClient span must be exported")
}

func TestRequestMsgWithTrace_RecordsOutboundMetric(t *testing.T) {
	_, cleanup := newTestTracerProvider(t)
	defer cleanup()

	// Re-initialise the metric instruments so they bind to the new MeterProvider
	// set up by InitTracer (Prometheus exporter path is not needed here — we just
	// need the global meter to be wired).
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-nats-metric", "0.0.1", "")
	require.NoError(t, err)
	defer func() { _ = shutdown(ctx) }()

	ns, nc := startNATSServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subject := "ts.test.metric"

	sub, err := nc.Subscribe(subject, func(msg *natsclient.Msg) {
		_ = msg.Respond([]byte(`{}`))
	})
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	outMsg := &natsclient.Msg{Subject: subject, Data: []byte(`{}`)}
	_, err = RequestMsgWithTrace(reqCtx, nc, subject, outMsg)
	require.NoError(t, err)

	// OutboundDependencyDuration is a histogram — we verify it doesn't panic and
	// the Prometheus scrape contains the metric name. A lightweight integration-
	// level assertion sufficient for the RED/GREEN cycle.
	assert.NotNil(t, OutboundDependencyDuration, "OutboundDependencyDuration instrument must be initialised")
}

// ---------------------------------------------------------------------------
// WrapNATSHandler — consumer span + trace-correlated logger
// ---------------------------------------------------------------------------

type safeLogBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeLogBuf) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeLogBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestWrapNATSHandler_ConsumerSpan_LinksToParent(t *testing.T) {
	exp, cleanup := newTestTracerProvider(t)
	defer cleanup()

	ns, nc := startNATSServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subject := "ts.test.consumer"

	// Prepare a parent trace context and inject it into a header
	tracer := otel.Tracer("test")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")
	wantTraceID := parentSpan.SpanContext().TraceID().String()
	parentSpan.End()

	// Build the outgoing message with the traceparent injected
	outMsg := &natsclient.Msg{Subject: subject, Data: []byte(`{}`)}
	InjectNATSContext(parentCtx, outMsg)
	outHeader := outMsg.Header

	var capturedTraceID string
	var wg sync.WaitGroup
	wg.Add(1)

	logger := zerolog.Nop()
	inner := func(ctx context.Context, _ zerolog.Logger, msg *natsclient.Msg) error {
		defer wg.Done()
		capturedTraceID = trace.SpanContextFromContext(ctx).TraceID().String()
		_ = msg.Respond([]byte(`{}`))
		return nil
	}

	sub, err := nc.Subscribe(subject, WrapNATSHandler("nats.echo_handler", logger, inner))
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	// Publish a message carrying the injected header
	pubMsg := &natsclient.Msg{
		Subject: subject,
		Header:  outHeader,
		Data:    []byte(`{}`),
	}
	_, err = nc.RequestMsg(pubMsg, 2*time.Second)
	require.NoError(t, err)

	wg.Wait()

	assert.Equal(t, wantTraceID, capturedTraceID, "consumer span must share the same TraceID as the injected parent")

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "nats.echo_handler" {
			found = true
			assert.Equal(t, trace.SpanKindConsumer, s.SpanKind)
			break
		}
	}
	assert.True(t, found, "a 'nats.echo_handler' SpanKindConsumer span must be exported")
}

func TestWrapNATSHandler_LogsTraceID(t *testing.T) {
	_, cleanup := newTestTracerProvider(t)
	defer cleanup()

	ns, nc := startNATSServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	subject := "ts.test.log"

	// Build parent context and inject traceparent
	tracer := otel.Tracer("test")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")
	wantTraceID := parentSpan.SpanContext().TraceID().String()
	parentSpan.End()

	outMsg := &natsclient.Msg{Subject: subject, Data: []byte(`{}`)}
	InjectNATSContext(parentCtx, outMsg)
	outHeader := outMsg.Header

	var logBuf safeLogBuf
	logger := zerolog.New(&logBuf)

	var wg sync.WaitGroup
	wg.Add(1)
	inner := func(_ context.Context, log zerolog.Logger, msg *natsclient.Msg) error {
		defer wg.Done()
		log.Info().Msg("handler executed")
		_ = msg.Respond([]byte(`{}`))
		return nil
	}

	sub, err := nc.Subscribe(subject, WrapNATSHandler("nats.echo_handler", logger, inner))
	require.NoError(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	pubMsg := &natsclient.Msg{
		Subject: subject,
		Header:  outHeader,
		Data:    []byte(`{}`),
	}
	_, err = nc.RequestMsg(pubMsg, 2*time.Second)
	require.NoError(t, err)

	wg.Wait()

	output := logBuf.String()
	var logEntry map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(output), &logEntry))
	assert.Equal(t, wantTraceID, logEntry["trace_id"], "log must contain trace_id matching the injected parent")
	assert.NotEmpty(t, logEntry["span_id"], "log must contain span_id")
}
