package bff_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/tacito-square/internal/bff"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

// Define mock structures for testing bootstrap
type mockSessionUseCase struct{}

func (m *mockSessionUseCase) InitiateLogin(ctx context.Context) (string, string, error) {
	return "", "", nil
}
func (m *mockSessionUseCase) HandleCallback(ctx context.Context, code, state string) (*model.Session, error) {
	return nil, nil
}
func (m *mockSessionUseCase) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return nil, nil
}
func (m *mockSessionUseCase) Logout(ctx context.Context, sessionID string) error { return nil }
func (m *mockSessionUseCase) BackchannelLogout(ctx context.Context, rawLogoutToken string) error {
	return nil
}
func (m *mockSessionUseCase) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return &model.Session{ID: sessionID, UserID: "mock-user", TenantID: "mock-tenant"}, nil
}

var _ inbound.SessionUseCase = (*mockSessionUseCase)(nil)

type mockEventStreamUseCase struct{}

func (m *mockEventStreamUseCase) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	return nil, nil
}

var _ inbound.EventStreamUseCase = (*mockEventStreamUseCase)(nil)

type mockSessionStore struct {
	pingErr error
}

func (m *mockSessionStore) Save(ctx context.Context, sess *model.Session, ttl time.Duration) error {
	return nil
}
func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	return nil, nil
}
func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error      { return nil }
func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID string) error { return nil }
func (m *mockSessionStore) DeleteByOIDCSessionID(ctx context.Context, issuer, oidcSessionID string) error {
	return nil
}
func (m *mockSessionStore) Ping(ctx context.Context) error { return m.pingErr }

var _ outbound.SessionStore = (*mockSessionStore)(nil)

type mockOIDCProvider struct{}

func (m *mockOIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) {
	return nil, nil
}
func (m *mockOIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) {
	return nil, nil
}
func (m *mockOIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) {
	return nil, nil
}
func (m *mockOIDCProvider) ValidateLogoutToken(ctx context.Context, rawToken string) (string, string, error) {
	return "", "", nil
}

var _ outbound.OIDCProvider = (*mockOIDCProvider)(nil)

type mockKeeperClient struct {
	pingErr error
}

func (m *mockKeeperClient) Ping(ctx context.Context) error { return m.pingErr }

var _ outbound.KeeperClient = (*mockKeeperClient)(nil)

func TestBFFServer_HealthzReturns200(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})
	require.NotNil(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "alive", body["status"])
}

func TestBFFServer_ReadyzReturns200_AllDepsHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	store := &mockSessionStore{pingErr: nil}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, store, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ready", body["status"])
}

func TestBFFServer_ReadyzReturns503_RedisFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	store := &mockSessionStore{pingErr: errors.New("redis dead")}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, store, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"checks"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body.Status)

	var redisCheckFound bool
	for _, check := range body.Checks {
		if check.Name == "redis" {
			redisCheckFound = true
			assert.Equal(t, "unhealthy", check.Status)
			assert.Contains(t, check.Error, "redis dead")
		}
	}
	assert.True(t, redisCheckFound)
}

func TestBFFServer_OpenAPIEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"openapi": "3.1.0"`)
	assert.Contains(t, w.Body.String(), `"version": "0.1.0"`)
}

func TestBFFServer_UIOpenAPIEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui/openapi.json", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"openapi": "3.1.0"`)
	assert.Contains(t, w.Body.String(), `"version": "0.1.0"`)
}

func TestBFFServer_RootWelcomePage_RedirectsToSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/ui/", w.Header().Get("Location"))
}

func TestBFFServer_RootWelcomePageWithSlash_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<title>Tacito Square BFF</title>")
	assert.Contains(t, w.Body.String(), "Piazza Tacito")
}

func TestBFFServer_IndexWelcomePage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui/index.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<title>Tacito Square BFF</title>")
	assert.Contains(t, w.Body.String(), "Piazza Tacito")
}

func TestBFFServer_SecureIndex_NoCookie_Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui/secure/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui/api/v1/auth/login", w.Header().Get("Location"))
}

func TestBFFServer_SecureIndex_WithCookie_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}

	mockSessionStoreInstance := &mockSessionStore{pingErr: nil}

	// Wait, bff.NewServer receives inbound.SessionUseCase as its second parameter

	// Let's verify what mock sessionUseCase mock is defined in bootstrap_test.go
	// Yes! The type is mockSessionUseCase. We can implement GetSession on mockSessionUseCase or configure a mock session.
	// Oh, wait, in bootstrap_test.go, mockSessionUseCase has GetSession returning (nil, nil) by default.
	// Let's define a mockSessionUseCase with custom GetSession behaviour if needed, but wait:
	// Let's check mockSessionUseCase in bootstrap_test.go (lines 23-30):
	// var _ inbound.SessionUseCase = (*mockSessionUseCase)(nil)
	// It's defined at package level in bootstrap_test.go, and it does not have customizable function fields like the mock in http_test.
	// Oh! It doesn't have fields for funcs. It just returns (nil, nil).
	// Let's check: if GetSession returns (nil, nil) without error, does the middleware treat it as a valid session?
	// Yes, err is nil, so it continues! It sets userID to sess.UserID (which is empty string but valid).
	// So passing &mockSessionUseCase{} as the session UseCase should successfully pass the middleware!
	// Let's write the test based on that.

	srv := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, mockSessionStoreInstance, &mockOIDCProvider{}, &mockKeeperClient{})

	req := httptest.NewRequest(http.MethodGet, "/ui/secure/index.html", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<title>Tacito Square BFF - Secure Zone</title>")
	assert.Contains(t, w.Body.String(), "Secure Logout")
}
