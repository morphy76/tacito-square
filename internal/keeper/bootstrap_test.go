package keeper

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_ReturnsGinEngine(t *testing.T) {
	srv := NewServer(nil, nil, nil)
	require.NotNil(t, srv)
}

func TestHealthz_Returns200(t *testing.T) {
	srv := NewServer(nil, nil, nil)

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
	srv := NewServer(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])

	checks, ok := body["checks"].([]interface{})
	require.True(t, ok)
	assert.Len(t, checks, 4)

	expectedNames := map[string]bool{
		"postgres":    true,
		"nats":        true,
		"redis":       true,
		"cache-redis": true,
	}

	for _, checkVal := range checks {
		checkMap, ok := checkVal.(map[string]interface{})
		require.True(t, ok)
		name := checkMap["name"].(string)
		assert.True(t, expectedNames[name])
	}
}

func TestOpenAPI_Returns200AndValidJSON(t *testing.T) {
	srv := NewServer(nil, nil, nil)

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
	srv := NewServer(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "go_goroutines")
}

func TestEndpoints_DatabaseUnavailable_Returns503(t *testing.T) {
	srv := NewServer(nil, nil, nil)

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

func TestNewServer_EventRoutesRegistered(t *testing.T) {
	srv := NewServer(nil, nil, nil)
	require.NotNil(t, srv)

	postEventFound := false
	getStreamFound := false
	for _, route := range srv.Routes() {
		if route.Method == http.MethodPost && route.Path == "/api/v1/events" {
			postEventFound = true
		}
		if route.Method == http.MethodGet && route.Path == "/api/v1/events/stream" {
			getStreamFound = true
		}
	}
	assert.True(t, postEventFound, "POST /api/v1/events route was not registered")
	assert.True(t, getStreamFound, "GET /api/v1/events/stream route was not registered")
}

func TestNewServer_EventsDatabaseUnavailable_Returns503(t *testing.T) {
	srv := NewServer(nil, nil, nil)
	require.NotNil(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`{"schema_ref":"urn:tacito:schema:conversational:start-thread:v1","payload":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-1")

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "Database service unavailable")
}
