package observability

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
)

var (
	// globalPrometheusExporter is the pull-based metrics exporter.
	globalPrometheusExporter *otelprometheus.Exporter

	// OTel Meter
	meter = otel.Meter("tacito-square")

	// HTTPRequestDuration collects request duration metrics.
	HTTPRequestDuration, _ = meter.Float64Histogram(
		"http_request_duration_seconds",
		otelmetric.WithDescription("Duration of HTTP requests in seconds."),
	)

	// HTTPRequestsTotal collects total request counts.
	HTTPRequestsTotal, _ = meter.Int64Counter(
		"http_requests_total",
		otelmetric.WithDescription("Total number of HTTP requests."),
	)

	// ActiveThreads collects the number of active processing threads.
	ActiveThreads, _ = meter.Int64UpDownCounter(
		"active_threads",
		otelmetric.WithDescription("Number of active processing threads."),
	)

	// AgentStatus collects the count of active agents by status.
	AgentStatus, _ = meter.Int64Gauge(
		"agent_status",
		otelmetric.WithDescription("Number of active agents by status."),
	)

	// PendingHITLCallbacks collects the number of pending HITL callbacks.
	PendingHITLCallbacks, _ = meter.Int64Gauge(
		"pending_hitl_callbacks",
		otelmetric.WithDescription("Number of pending HITL callbacks."),
	)

	// CommunityQuotaUtilization collects community quota utilization.
	CommunityQuotaUtilization, _ = meter.Int64Gauge(
		"community_quota_utilization",
		otelmetric.WithDescription("Quota utilization per community."),
	)

	// AgentQuotaUtilization collects agent quota utilization.
	AgentQuotaUtilization, _ = meter.Int64Gauge(
		"agent_quota_utilization",
		otelmetric.WithDescription("Quota utilization per agent."),
	)

	// OutboundDependencyDuration collects request durations to external services.
	OutboundDependencyDuration, _ = meter.Float64Histogram(
		"outbound_dependency_duration_seconds",
		otelmetric.WithDescription("Duration of outbound request to external dependencies in seconds."),
	)

	// AgentNATSMessagesProcessedTotal collects total NATS messages processed.
	AgentNATSMessagesProcessedTotal, _ = meter.Int64Counter(
		"agent_nats_messages_processed_total",
		otelmetric.WithDescription("Total NATS messages processed by the agent."),
	)

	// AgentNATSProcessingDuration collects processing durations for NATS messages.
	AgentNATSProcessingDuration, _ = meter.Float64Histogram(
		"agent_nats_processing_duration_seconds",
		otelmetric.WithDescription("Time taken to process and reply to a NATS message."),
	)

	// AgentSTMOperationsTotal collects total short-term memory operations.
	AgentSTMOperationsTotal, _ = meter.Int64Counter(
		"agent_stm_operations_total",
		otelmetric.WithDescription("Total short-term memory operations."),
	)

	// AgentSTMOperationDuration collects STM operation durations.
	AgentSTMOperationDuration, _ = meter.Float64Histogram(
		"agent_stm_operation_duration_seconds",
		otelmetric.WithDescription("Short-term memory operation duration."),
	)

	// AgentLTMOperationsTotal collects total long-term memory operations.
	AgentLTMOperationsTotal, _ = meter.Int64Counter(
		"agent_ltm_operations_total",
		otelmetric.WithDescription("Total long-term memory operations."),
	)

	// AgentLTMOperationDuration collects LTM operation durations.
	AgentLTMOperationDuration, _ = meter.Float64Histogram(
		"agent_ltm_operation_duration_seconds",
		otelmetric.WithDescription("Long-term memory operation duration."),
	)

	// AgentBrainRequestsTotal collects total brain requests.
	AgentBrainRequestsTotal, _ = meter.Int64Counter(
		"agent_brain_requests_total",
		otelmetric.WithDescription("Total execution requests dispatched to the LLM backend."),
	)

	// AgentBrainRequestDuration collects brain request durations.
	AgentBrainRequestDuration, _ = meter.Float64Histogram(
		"agent_brain_request_duration_seconds",
		otelmetric.WithDescription("Duration of brain request to LLM backend."),
	)

	// AgentBrainTokensTotal collects total brain tokens.
	AgentBrainTokensTotal, _ = meter.Int64Counter(
		"agent_brain_tokens_total",
		otelmetric.WithDescription("Total brain tokens processed."),
	)
)

// initInstruments initializes all OpenTelemetry metrics instruments.
// This is called during telemetry setup to ensure instruments are registered
// with the fully-configured concrete MeterProvider.
func initInstruments() {
	meter = otel.Meter("tacito-square")

	HTTPRequestDuration, _ = meter.Float64Histogram(
		"http_request_duration_seconds",
		otelmetric.WithDescription("Duration of HTTP requests in seconds."),
	)

	HTTPRequestsTotal, _ = meter.Int64Counter(
		"http_requests_total",
		otelmetric.WithDescription("Total number of HTTP requests."),
	)

	ActiveThreads, _ = meter.Int64UpDownCounter(
		"active_threads",
		otelmetric.WithDescription("Number of active processing threads."),
	)

	AgentStatus, _ = meter.Int64Gauge(
		"agent_status",
		otelmetric.WithDescription("Number of active agents by status."),
	)

	PendingHITLCallbacks, _ = meter.Int64Gauge(
		"pending_hitl_callbacks",
		otelmetric.WithDescription("Number of pending HITL callbacks."),
	)

	CommunityQuotaUtilization, _ = meter.Int64Gauge(
		"community_quota_utilization",
		otelmetric.WithDescription("Quota utilization per community."),
	)

	AgentQuotaUtilization, _ = meter.Int64Gauge(
		"agent_quota_utilization",
		otelmetric.WithDescription("Quota utilization per agent."),
	)

	OutboundDependencyDuration, _ = meter.Float64Histogram(
		"outbound_dependency_duration_seconds",
		otelmetric.WithDescription("Duration of outbound request to external dependencies in seconds."),
	)

	AgentNATSMessagesProcessedTotal, _ = meter.Int64Counter(
		"agent_nats_messages_processed_total",
		otelmetric.WithDescription("Total NATS messages processed by the agent."),
	)

	AgentNATSProcessingDuration, _ = meter.Float64Histogram(
		"agent_nats_processing_duration_seconds",
		otelmetric.WithDescription("Time taken to process and reply to a NATS message."),
	)

	AgentSTMOperationsTotal, _ = meter.Int64Counter(
		"agent_stm_operations_total",
		otelmetric.WithDescription("Total short-term memory operations."),
	)

	AgentSTMOperationDuration, _ = meter.Float64Histogram(
		"agent_stm_operation_duration_seconds",
		otelmetric.WithDescription("Short-term memory operation duration."),
	)

	AgentLTMOperationsTotal, _ = meter.Int64Counter(
		"agent_ltm_operations_total",
		otelmetric.WithDescription("Total long-term memory operations."),
	)

	AgentLTMOperationDuration, _ = meter.Float64Histogram(
		"agent_ltm_operation_duration_seconds",
		otelmetric.WithDescription("Long-term memory operation duration."),
	)

	AgentBrainRequestsTotal, _ = meter.Int64Counter(
		"agent_brain_requests_total",
		otelmetric.WithDescription("Total execution requests dispatched to the LLM backend."),
	)

	AgentBrainRequestDuration, _ = meter.Float64Histogram(
		"agent_brain_request_duration_seconds",
		otelmetric.WithDescription("Duration of brain request to LLM backend."),
	)

	AgentBrainTokensTotal, _ = meter.Int64Counter(
		"agent_brain_tokens_total",
		otelmetric.WithDescription("Total brain tokens processed."),
	)
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

		ctx := c.Request.Context()
		attrs := otelmetric.WithAttributes(
			attribute.String("path", path),
			attribute.String("method", method),
			attribute.String("status", status),
		)

		HTTPRequestDuration.Record(ctx, duration, attrs)
		HTTPRequestsTotal.Add(ctx, 1, attrs)
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

	dbMeter := otel.Meter("db-pool")

	_, _ = dbMeter.Int64ObservableGauge(
		"db_pool_acquired_connections",
		otelmetric.WithDescription("Number of currently acquired/active connections in the database pool."),
		otelmetric.WithInt64Callback(func(ctx context.Context, obs otelmetric.Int64Observer) error {
			stats := pool.Stat()
			obs.Observe(int64(stats.AcquiredConns()))
			return nil
		}),
	)

	_, _ = dbMeter.Int64ObservableGauge(
		"db_pool_idle_connections",
		otelmetric.WithDescription("Number of currently idle connections in the database pool."),
		otelmetric.WithInt64Callback(func(ctx context.Context, obs otelmetric.Int64Observer) error {
			stats := pool.Stat()
			obs.Observe(int64(stats.IdleConns()))
			return nil
		}),
	)

	_, _ = dbMeter.Int64ObservableGauge(
		"db_pool_total_connections",
		otelmetric.WithDescription("Total number of connections currently in the database pool."),
		otelmetric.WithInt64Callback(func(ctx context.Context, obs otelmetric.Int64Observer) error {
			stats := pool.Stat()
			obs.Observe(int64(stats.TotalConns()))
			return nil
		}),
	)

	_, _ = dbMeter.Int64ObservableGauge(
		"db_pool_max_connections",
		otelmetric.WithDescription("Maximum number of connections allowed in the database pool."),
		otelmetric.WithInt64Callback(func(ctx context.Context, obs otelmetric.Int64Observer) error {
			stats := pool.Stat()
			obs.Observe(int64(stats.MaxConns()))
			return nil
		}),
	)
}
