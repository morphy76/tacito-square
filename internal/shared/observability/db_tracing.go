package observability

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type dbStartTimeKeyType struct{}

var dbStartTimeKey = dbStartTimeKeyType{}

// PgxQueryTracer implements the pgx.QueryTracer interface to trace PostgreSQL queries.
type PgxQueryTracer struct {
	tracer trace.Tracer
}

// NewPgxQueryTracer creates a new PgxQueryTracer.
func NewPgxQueryTracer() *PgxQueryTracer {
	return &PgxQueryTracer{
		tracer: otel.Tracer("postgres"),
	}
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls.
func (t *PgxQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "db.query", trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.statement", data.SQL),
	)

	// Inject the starting time into the context for query latency metric measurements
	return context.WithValue(ctx, dbStartTimeKey, time.Now())
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls.
func (t *PgxQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	defer span.End()

	// 1. Set OpenTelemetry span status and errors
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// 2. Measure and record database query duration metrics
	if startTime, ok := ctx.Value(dbStartTimeKey).(time.Time); ok {
		duration := time.Since(startTime).Seconds()
		status := "success"
		if data.Err != nil {
			status = "failure"
		}
		OutboundDependencyDuration.WithLabelValues("postgresql", "query", status).Observe(duration)
	}
}
