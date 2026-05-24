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
