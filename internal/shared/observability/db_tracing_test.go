package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestPgxQueryTracer_Telemetry(t *testing.T) {
	// Setup unified OpenTelemetry telemetry
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-db-service", "1.0.0", "")
	require.NoError(t, err)
	defer func() { _ = shutdown(ctx) }()

	// Setup mock OpenTelemetry tracer provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Create pgx tracer
	pgTracer := NewPgxQueryTracer()
	assert.NotNil(t, pgTracer)

	// 1. Trace Query Start
	startData := pgx.TraceQueryStartData{
		SQL: "SELECT * FROM agents WHERE id = $1",
	}
	ctx = pgTracer.TraceQueryStart(ctx, nil, startData)
	assert.NotNil(t, ctx)

	// Verify start time is stored in context
	startTime, ok := ctx.Value(dbStartTimeKey).(time.Time)
	assert.True(t, ok, "start time should be stored in context")
	assert.WithinDuration(t, time.Now(), startTime, 50*time.Millisecond)

	// 2. Trace Query End
	endData := pgx.TraceQueryEndData{
		Err: nil,
	}

	pgTracer.TraceQueryEnd(ctx, nil, endData)

	// Verify OTel outbound dependency metric is captured in the metrics handler
	r := gin.New()
	r.GET("/metrics", MetricsHandler())

	reqMetrics, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	wMetrics := httptest.NewRecorder()
	r.ServeHTTP(wMetrics, reqMetrics)

	assert.Equal(t, http.StatusOK, wMetrics.Code)
	body := wMetrics.Body.String()

	// Assert that OTel outbound dependency metric appears in scrape
	assert.Contains(t, body, "outbound_dependency_duration_seconds")
}
