// Package observability provides structured logging and tracing setup
// for Tacito Square components using zerolog and OpenTelemetry.
package observability

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// LogClaimsKeys defines which token claims are included in log output.
// This is configured at build time via ldflags or compile-time constants.
var LogClaimsKeys = []string{"sub", "email"}

// NewLogger creates a structured zerolog.Logger with the given level.
func NewLogger(level string, w io.Writer) zerolog.Logger {
	if w == nil {
		w = os.Stdout
	}

	var lvl zerolog.Level
	switch level {
	case "debug":
		lvl = zerolog.DebugLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	case "trace":
		lvl = zerolog.TraceLevel
	default:
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(w).Level(lvl).With().Timestamp().Logger()
}

// WithTraceID returns a sub-logger enriched with the trace_id and span_id
// from the current OpenTelemetry span context.
func WithTraceID(logger zerolog.Logger, spanCtx trace.SpanContext) zerolog.Logger {
	if spanCtx.HasTraceID() {
		logger = logger.With().
			Str("trace_id", spanCtx.TraceID().String()).
			Str("span_id", spanCtx.SpanID().String()).
			Logger()
	}
	return logger
}

// WithClaims returns a sub-logger enriched with the configured token claims.
// Only claims whose keys match LogClaimsKeys are included.
func WithClaims(logger zerolog.Logger, claims map[string]interface{}) zerolog.Logger {
	ctx := logger.With()
	for _, key := range LogClaimsKeys {
		if val, ok := claims[key]; ok {
			ctx = ctx.Interface(key, val)
		}
	}
	return ctx.Logger()
}
