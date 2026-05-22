package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTracer_EmptyEndpoint_ReturnsNoopShutdown(t *testing.T) {
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "0.1.0", "")

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// No-op shutdown should succeed without error
	err = shutdown(ctx)
	assert.NoError(t, err)
}

func TestTracer_ReturnsValidTracer(t *testing.T) {
	tracer := Tracer("test-component")
	assert.NotNil(t, tracer)
}

func TestInitTracer_WithEndpoint_ReturnsShutdown(t *testing.T) {
	// Use a dummy endpoint — the exporter won't connect but InitTracer should succeed.
	// In real code this would connect to an OTel collector.
	ctx := context.Background()
	shutdown, err := InitTracer(ctx, "test-service", "0.1.0", "localhost:4317")

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown should not error even without a real collector
	err = shutdown(ctx)
	assert.NoError(t, err)
}
