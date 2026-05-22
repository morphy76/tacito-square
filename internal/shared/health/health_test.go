package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLivezHandler_AlwaysReturns200(t *testing.T) {
	probe := NewProbe(time.Second)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	probe.LivezHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "alive")
}

func TestReadyzHandler_AllHealthy(t *testing.T) {
	probe := NewProbe(time.Second,
		PingChecker("db", func(ctx context.Context) error { return nil }),
		PingChecker("nats", func(ctx context.Context) error { return nil }),
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	probe.ReadyzHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ReadinessResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
	assert.Len(t, result.Checks, 2)
	for _, c := range result.Checks {
		assert.Equal(t, "healthy", c.Status)
	}
}

func TestReadyzHandler_OneUnhealthy(t *testing.T) {
	probe := NewProbe(time.Second,
		PingChecker("db", func(ctx context.Context) error { return nil }),
		PingChecker("redis", func(ctx context.Context) error {
			return fmt.Errorf("connection refused")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	probe.ReadyzHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var result ReadinessResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", result.Status)

	// Find the unhealthy check
	var unhealthy *CheckResult
	for _, c := range result.Checks {
		if c.Name == "redis" {
			unhealthy = &c
		}
	}
	require.NotNil(t, unhealthy)
	assert.Equal(t, "unhealthy", unhealthy.Status)
	assert.Contains(t, unhealthy.Error, "connection refused")
}

func TestReadyzHandler_NoCheckers(t *testing.T) {
	probe := NewProbe(time.Second)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	probe.ReadyzHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPChecker_HealthyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPChecker("llm", server.URL)
	err := checker.Check(context.Background())

	assert.NoError(t, err)
}

func TestHTTPChecker_UnhealthyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := HTTPChecker("llm", server.URL)
	err := checker.Check(context.Background())

	assert.Error(t, err)
}
