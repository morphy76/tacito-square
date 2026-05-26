package operator_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morphy76/tacito-square/internal/operator"
	_ "github.com/morphy76/tacito-square/internal/operator/adapters/inbound"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_ReturnsGinEngine(t *testing.T) {
	srv := operator.NewServer()
	require.NotNil(t, srv)
}

func TestHealthz_Returns200(t *testing.T) {
	srv := operator.NewServer()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "alive", body["status"])
}

func TestReadyz_Returns200(t *testing.T) {
	healthyChecker := health.Checker{
		Name: "test-healthy",
		Check: func(ctx context.Context) error {
			return nil
		},
	}
	srv := operator.NewServer(healthyChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ready", body["status"])
}

func TestReadyz_Returns503(t *testing.T) {
	unhealthyChecker := health.Checker{
		Name: "test-unhealthy",
		Check: func(ctx context.Context) error {
			return errors.New("connection failed to test-unhealthy")
		},
	}
	srv := operator.NewServer(unhealthyChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])
}

func TestMetrics_ExposesCustomMetrics(t *testing.T) {
	srv := operator.NewServer()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()

	// Assert that standard reconciler metrics are exposed
	assert.Contains(t, body, "tacito_operator_reconciliation_total")
	assert.Contains(t, body, "tacito_operator_reconciliation_duration_seconds")
	assert.Contains(t, body, "tacito_operator_active_agents")
}
