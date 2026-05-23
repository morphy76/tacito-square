package keeper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_ReturnsGinEngine(t *testing.T) {
	srv := NewServer(nil)
	require.NotNil(t, srv)
}

func TestHealthz_Returns200(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "alive", body["status"])
}

func TestReadyz_Returns503_WhenDatabaseUnavailable(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])
}

func TestOpenAPI_Returns200AndValidJSON(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "3.0.3", body["openapi"])

	info, ok := body["info"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Keeper API", info["title"])
}

func TestOpenAPISpec_MatchesCommittedContract(t *testing.T) {
	contractPath := "../../api/openapi/openapi.json"
	contractBytes, err := os.ReadFile(contractPath)
	require.NoError(t, err, "failed to read committed contract at %s", contractPath)

	var embeddedSpec map[string]interface{}
	err = json.Unmarshal(openapiJSON, &embeddedSpec)
	require.NoError(t, err)

	var committedSpec map[string]interface{}
	err = json.Unmarshal(contractBytes, &committedSpec)
	require.NoError(t, err)

	assert.Equal(t, committedSpec, embeddedSpec, "Embedded OpenAPI spec does not match the committed contract in api/openapi/openapi.json")
}

func TestMetrics_Returns200AndPrometheusFormat(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "go_goroutines")
}

func TestEndpoints_DatabaseUnavailable_Returns503(t *testing.T) {
	srv := NewServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/llm-bindings", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-Subscription-ID", "sub-1")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Database service unavailable", body["error"])
}
