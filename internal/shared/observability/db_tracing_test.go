package observability

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestPgxQueryTracer_Telemetry(t *testing.T) {
	// Setup mock OpenTelemetry tracer provider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	// Create pgx tracer
	pgTracer := NewPgxQueryTracer()
	assert.NotNil(t, pgTracer)

	ctx := context.Background()

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

	// We reset the histogram vector to get a clean state
	OutboundDependencyDuration.Reset()

	pgTracer.TraceQueryEnd(ctx, nil, endData)

	// Verify Prometheus metric was captured
	metricChan := make(chan prometheus.Metric, 10)
	OutboundDependencyDuration.Collect(metricChan)
	close(metricChan)

	var foundMetric bool
	for m := range metricChan {
		var metric dto.Metric
		err := m.Write(&metric)
		assert.NoError(t, err)

		// Assert labels
		labels := metric.GetLabel()
		assert.Len(t, labels, 3)

		var hasPostgresLabel, hasQueryLabel, hasSuccessLabel bool
		for _, l := range labels {
			if l.GetName() == "dependency" && l.GetValue() == "postgresql" {
				hasPostgresLabel = true
			}
			if l.GetName() == "operation" && l.GetValue() == "query" {
				hasQueryLabel = true
			}
			if l.GetName() == "status" && l.GetValue() == "success" {
				hasSuccessLabel = true
			}
		}

		if hasPostgresLabel && hasQueryLabel && hasSuccessLabel {
			foundMetric = true
			assert.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
		}
	}

	assert.True(t, foundMetric, "Prometheus outbound dependency metric for postgres should be recorded")
}
