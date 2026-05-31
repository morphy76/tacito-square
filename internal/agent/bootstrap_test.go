package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_ReturnsGinEngine(t *testing.T) {
	srv := NewServer()
	require.NotNil(t, srv)
}

func TestHealthz_Returns200(t *testing.T) {
	srv := NewServer()

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
	srv := NewServer()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ready", body["status"])
}

func TestReadyz_Returns503_WhenDependencyUnhealthy(t *testing.T) {
	// Register a failing mock checker to represent RED phase of health probe
	failingChecker := health.Checker{
		Name: "redis",
		Check: func(ctx context.Context) error {
			return errors.New("redis connection refused")
		},
	}
	srv := NewServer(failingChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])
}

func TestReadyz_Returns503_WhenQdrantUnhealthy(t *testing.T) {
	qdrantChecker := health.Checker{
		Name: "qdrant",
		Check: func(ctx context.Context) error {
			return errors.New("qdrant connection refused")
		},
	}
	srv := NewServer(qdrantChecker)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])

	checks := body["checks"].([]interface{})
	foundQdrant := false
	for _, ch := range checks {
		chMap := ch.(map[string]interface{})
		if chMap["name"] == "qdrant" {
			assert.Equal(t, "unhealthy", chMap["status"])
			assert.Equal(t, "qdrant connection refused", chMap["error"])
			foundQdrant = true
		}
	}
	assert.True(t, foundQdrant)
}
