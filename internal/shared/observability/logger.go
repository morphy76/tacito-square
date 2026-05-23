// Package observability provides structured logging and tracing setup
// for Tacito Square components using zerolog and OpenTelemetry.
package observability

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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

// LoggingMiddleware returns a Gin middleware that logs every request in structured JSON,
// enriched with the active OpenTelemetry trace and span contexts when available.
func LoggingMiddleware() gin.HandlerFunc {
	logger := NewLogger("info", os.Stdout)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Skip health probes and metrics to avoid noisy logs in production
		if path == "/metrics" || path == "/healthz" || path == "/readyz" {
			return
		}

		spanCtx := trace.SpanContextFromContext(c.Request.Context())
		reqLogger := WithTraceID(logger, spanCtx)

		reqLogger.Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("duration_ms", duration).
			Msg("HTTP request processed")
	}
}
