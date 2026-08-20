package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestMetricsMiddleware_InstrumentsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(MetricsMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterDBPoolStats_NilPool_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RegisterDBPoolStats(nil)
	})
}

func TestMetricsExposition_OTel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	// Initialize unified telemetry (InitTracer acts as our telemetry bootstrapper)
	shutdown, err := InitTracer(ctx, "test-service", "1.0.0", "")
	require.NoError(t, err)
	defer func() { _ = shutdown(ctx) }()

	r := gin.New()
	r.Use(MetricsMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/metrics", MetricsHandler())

	// Record a request metric
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Scrape the /metrics endpoint
	reqMetrics, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	wMetrics := httptest.NewRecorder()
	r.ServeHTTP(wMetrics, reqMetrics)

	assert.Equal(t, http.StatusOK, wMetrics.Code)
	body := wMetrics.Body.String()

	// Assert that OTel registered metrics appear in the Prometheus scrape
	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds")
	
	// Assert that we have a global MeterProvider set
	assert.NotNil(t, otel.GetMeterProvider())
}
