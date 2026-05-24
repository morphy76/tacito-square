package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestNewLogger_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", &buf)

	logger.Info().Str("key", "value").Msg("test message")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "test message", entry["message"])
	assert.Equal(t, "value", entry["key"])
	assert.Equal(t, "info", entry["level"])
}

func TestNewLogger_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("debug", &buf)

	logger.Debug().Msg("debug msg")

	assert.Contains(t, buf.String(), "debug msg")
}

func TestNewLogger_DefaultsToInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("unknown", &buf)

	logger.Debug().Msg("should not appear")
	assert.Empty(t, buf.String())

	logger.Info().Msg("should appear")
	assert.NotEmpty(t, buf.String())
}

func TestWithTraceID_AddsTraceFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", &buf)

	// Create a valid trace ID and span ID
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	enriched := WithTraceID(logger, spanCtx)
	enriched.Info().Msg("traced")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", entry["trace_id"])
	assert.Equal(t, "00f067aa0ba902b7", entry["span_id"])
}

func TestWithTraceID_NoTraceID_NoFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", &buf)

	enriched := WithTraceID(logger, trace.SpanContext{})
	enriched.Info().Msg("no trace")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Nil(t, entry["trace_id"])
}

func TestWithClaims_AddsConfiguredClaims(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", &buf)

	// Override configured claim keys for test
	origKeys := LogClaimsKeys
	LogClaimsKeys = []string{"sub", "email", "org"}
	defer func() { LogClaimsKeys = origKeys }()

	claims := map[string]interface{}{
		"sub":   "user-123",
		"email": "user@example.com",
		"org":   "acme",
		"iat":   1234567890, // should NOT appear
	}

	enriched := WithClaims(logger, claims)
	enriched.Info().Msg("with claims")

	var entry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, "user-123", entry["sub"])
	assert.Equal(t, "user@example.com", entry["email"])
	assert.Equal(t, "acme", entry["org"])
	assert.Nil(t, entry["iat"]) // not in LogClaimsKeys
}

func TestWithContext_EnrichesLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("info", &buf)

	ten, err := tenant.New("acme.com", "sub-1")
	require.NoError(t, err)

	ctx := tenant.ContextWithTenant(context.Background(), ten)

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})
	ctx = trace.ContextWithSpanContext(ctx, spanCtx)

	enriched := WithContext(logger, ctx)
	enriched.Info().Msg("context enriched log")

	var entry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)

	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", entry["trace_id"])
	assert.Equal(t, "00f067aa0ba902b7", entry["span_id"])
	assert.Equal(t, "acme.com-sub-1", entry["tenant_id"])
	assert.Equal(t, "context enriched log", entry["message"])
}
