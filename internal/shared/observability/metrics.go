package observability

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

// RegisterDBPoolStats registers database connection pool metrics for the given pool.
func RegisterDBPoolStats(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	prometheus.MustRegister(NewDBPoolCollector(pool))
}

type dbPoolCollector struct {
	pool          *pgxpool.Pool
	acquiredConns *prometheus.Desc
	idleConns     *prometheus.Desc
	totalConns    *prometheus.Desc
	maxConns      *prometheus.Desc
}

// NewDBPoolCollector creates a new custom Prometheus collector for the database connection pool.
func NewDBPoolCollector(pool *pgxpool.Pool) prometheus.Collector {
	return &dbPoolCollector{
		pool: pool,
		acquiredConns: prometheus.NewDesc(
			"db_pool_acquired_connections",
			"Number of currently acquired/active connections in the database pool.",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			"db_pool_idle_connections",
			"Number of currently idle connections in the database pool.",
			nil, nil,
		),
		totalConns: prometheus.NewDesc(
			"db_pool_total_connections",
			"Total number of connections currently in the database pool.",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			"db_pool_max_connections",
			"Maximum number of connections allowed in the database pool.",
			nil, nil,
		),
	}
}

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquiredConns
	ch <- c.idleConns
	ch <- c.totalConns
	ch <- c.maxConns
}

func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool == nil {
		return
	}
	stats := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.totalConns, prometheus.GaugeValue, float64(stats.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()))
}
