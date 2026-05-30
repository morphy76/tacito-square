package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer_EmptyEndpoint_ReturnsNoopShutdown(t *testing.T) {
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "0.1.0", "")

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// No-op shutdown should succeed without error
	err = shutdown(ctx)
	assert.NoError(t, err)
}

func TestTracer_ReturnsValidTracer(t *testing.T) {
	tracer := Tracer("test-component")
	assert.NotNil(t, tracer)
}

func TestInitTracer_WithEndpoint_ReturnsShutdown(t *testing.T) {
	// Use a dummy endpoint — the exporter won't connect but InitTracer should succeed.
	// In real code this would connect to an OTel collector.
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "0.1.0", "localhost:4317")

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown should not error even without a real collector
	err = shutdown(ctx)
	assert.NoError(t, err)
}

func TestTracingMiddleware_ExtractsTraceContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up tracer provider and global propagator first
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "0.1.0", "localhost:4317")
	require.NoError(t, err)
	defer shutdown(ctx)

	r := gin.New()
	r.Use(TracingMiddleware("test-service"))

	var extractedSpanCtx trace.SpanContext
	r.GET("/test", func(c *gin.Context) {
		extractedSpanCtx = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// W3C traceparent format: version-traceID-parentID-traceFlags
	// Let's create a dummy trace ID: 4bf92f3577b34da6a3ce929d0e0e4736
	// Parent ID: 00f067aa0ba902b7
	dummyTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("traceparent", dummyTraceParent)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, extractedSpanCtx.IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", extractedSpanCtx.TraceID().String())
}

func TestTracingMiddleware_BypassesSystemEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up tracer provider first
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service-bypass", "0.1.0", "localhost:4317")
	require.NoError(t, err)
	defer shutdown(ctx)

	r := gin.New()
	r.Use(TracingMiddleware("test-service-bypass"))

	var extractedSpanCtx trace.SpanContext

	r.GET("/healthz", func(c *gin.Context) {
		extractedSpanCtx = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	r.GET("/readyz", func(c *gin.Context) {
		extractedSpanCtx = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	r.GET("/metrics", func(c *gin.Context) {
		extractedSpanCtx = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})
	r.GET("/api/v1/echo", func(c *gin.Context) {
		extractedSpanCtx = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// Test 1: Verify healthz does not generate a span when no parent is present
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.False(t, extractedSpanCtx.IsValid(), "healthz should not have a valid span context when no parent is supplied")

	// Test 2: Verify readyz does not generate a span when no parent is present
	req, _ = http.NewRequest(http.MethodGet, "/readyz", nil)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.False(t, extractedSpanCtx.IsValid(), "readyz should not have a valid span context when no parent is supplied")

	// Test 3: Verify metrics does not generate a span when no parent is present
	req, _ = http.NewRequest(http.MethodGet, "/metrics", nil)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.False(t, extractedSpanCtx.IsValid(), "metrics should not have a valid span context when no parent is supplied")

	// Test 4: Verify that a normal API route DOES generate a new span
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/echo", nil)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, extractedSpanCtx.IsValid(), "normal api endpoints should have a valid generated span context")

	// Test 5: Verify healthz propagates parent trace context but does NOT create a new child span
	dummyTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	req, _ = http.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("traceparent", dummyTraceParent)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, extractedSpanCtx.IsValid(), "healthz should propagate valid parent context")
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", extractedSpanCtx.TraceID().String())
	assert.Equal(t, "00f067aa0ba902b7", extractedSpanCtx.SpanID().String(), "span ID should match the parent unchanged (no new child span started)")

	// Test 6: Verify normal endpoint propagates parent context AND generates a new child span
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/echo", nil)
	req.Header.Set("traceparent", dummyTraceParent)
	resp = httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.True(t, extractedSpanCtx.IsValid())
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", extractedSpanCtx.TraceID().String())
	assert.NotEqual(t, "00f067aa0ba902b7", extractedSpanCtx.SpanID().String(), "normal endpoint must generate a new child span ID different from parent")
}
