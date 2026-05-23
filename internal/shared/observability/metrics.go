package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTPRequestDuration collects request duration metrics.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method", "status"},
	)

	// HTTPRequestsTotal collects total request counts.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"path", "method", "status"},
	)
)

func init() {
	// Register the technical metrics
	prometheus.MustRegister(HTTPRequestDuration)
	prometheus.MustRegister(HTTPRequestsTotal)
}

// MetricsMiddleware auto-instruments HTTP metrics for all routes.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		// Avoid collecting metrics for /metrics and health routes to keep it clean and simple
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			return
		}

		HTTPRequestDuration.WithLabelValues(path, method, status).Observe(duration)
		HTTPRequestsTotal.WithLabelValues(path, method, status).Inc()
	}
}

// MetricsHandler wraps promhttp.Handler for Gin.
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
