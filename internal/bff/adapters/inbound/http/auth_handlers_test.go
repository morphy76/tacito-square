package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	bffhttp "github.com/morphy76/tacito-square/internal/bff/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

type mockSessionUseCase struct {
	InitiateLoginFunc     func(ctx context.Context) (string, string, error)
	HandleCallbackFunc    func(ctx context.Context, code, state string) (*model.Session, error)
	RefreshSessionFunc    func(ctx context.Context, sessionID string) (*model.Session, error)
	LogoutFunc            func(ctx context.Context, sessionID string) error
	BackchannelLogoutFunc func(ctx context.Context, rawLogoutToken string) error
	GetSessionFunc        func(ctx context.Context, sessionID string) (*model.Session, error)
}

func (m *mockSessionUseCase) InitiateLogin(ctx context.Context) (string, string, error) {
	if m.InitiateLoginFunc != nil {
		return m.InitiateLoginFunc(ctx)
	}
	return "", "", nil
}

func (m *mockSessionUseCase) HandleCallback(ctx context.Context, code, state string) (*model.Session, error) {
	if m.HandleCallbackFunc != nil {
		return m.HandleCallbackFunc(ctx, code, state)
	}
	return nil, nil
}

func (m *mockSessionUseCase) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if m.RefreshSessionFunc != nil {
		return m.RefreshSessionFunc(ctx, sessionID)
	}
	return nil, nil
}

func (m *mockSessionUseCase) Logout(ctx context.Context, sessionID string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionUseCase) BackchannelLogout(ctx context.Context, rawLogoutToken string) error {
	if m.BackchannelLogoutFunc != nil {
		return m.BackchannelLogoutFunc(ctx, rawLogoutToken)
	}
	return nil
}

func (m *mockSessionUseCase) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx, sessionID)
	}
	return nil, nil
}

var _ inbound.SessionUseCase = (*mockSessionUseCase)(nil)

func TestAuthHandler_Login_RedirectsToOIDC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		InitiateLoginFunc: func(ctx context.Context) (string, string, error) {
			return "https://oidc-provider.com/auth?state=xyz", "xyz", nil
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/api/v1/auth/login", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://oidc-provider.com/auth?state=xyz", w.Header().Get("Location"))

	// Verify state cookie is set
	cookies := w.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "bff_oidc_state" {
			stateCookie = c
		}
	}
	assert.NotNil(t, stateCookie)
	assert.Equal(t, "xyz", stateCookie.Value)
	assert.True(t, stateCookie.HttpOnly)
	assert.Equal(t, "/ui", stateCookie.Path)
}

func TestAuthHandler_Callback_Success_SetsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	now := time.Now().UTC()
	sess := &model.Session{
		ID:                   "session-123",
		UserID:               "user-456",
		TenantID:             "tenant-789",
		AccessTokenExpiresAt: now.Add(1 * time.Hour),
	}

	mockUC := &mockSessionUseCase{
		HandleCallbackFunc: func(ctx context.Context, code, state string) (*model.Session, error) {
			assert.Equal(t, "code-abc", code)
			assert.Equal(t, "state-xyz", state)
			return sess, nil
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/api/v1/auth/callback?code=code-abc&state=state-xyz", nil)
	// Set state cookie to match incoming state
	req.AddCookie(&http.Cookie{
		Name:  "bff_oidc_state",
		Value: "state-xyz",
		Path:  "/ui",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui", w.Header().Get("Location"))

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	var clearedStateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "bff_session_id" {
			sessionCookie = c
		}
		if c.Name == "bff_oidc_state" {
			clearedStateCookie = c
		}
	}
	assert.NotNil(t, sessionCookie)
	assert.Equal(t, "session-123", sessionCookie.Value)
	assert.Equal(t, "/ui", sessionCookie.Path)
	assert.True(t, sessionCookie.HttpOnly)
	assert.True(t, sessionCookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, sessionCookie.SameSite)

	assert.NotNil(t, clearedStateCookie)
	assert.True(t, clearedStateCookie.Expires.Before(time.Now()))
	assert.Equal(t, "/ui", clearedStateCookie.Path)
}

func TestAuthHandler_Callback_ExchangeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		HandleCallbackFunc: func(ctx context.Context, code, state string) (*model.Session, error) {
			return nil, errors.New("exchange error")
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/api/v1/auth/callback?code=code-abc&state=state-xyz", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_oidc_state",
		Value: "state-xyz",
		Path:  "/ui",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "exchange error", resp["error"])
}

func TestAuthHandler_Logout_ClearsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var calledSessionID string
	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return &model.Session{
				ID:       "session-123",
				TenantID: "tenant-789",
			}, nil
		},
		LogoutFunc: func(ctx context.Context, sessionID string) error {
			calledSessionID = sessionID
			return nil
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ui/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
		Path:  "/ui",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui", w.Header().Get("Location"))
	assert.Equal(t, "session-123", calledSessionID)

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "bff_session_id" {
			sessionCookie = c
		}
	}
	assert.NotNil(t, sessionCookie)
	assert.True(t, sessionCookie.Expires.Before(time.Now()))
	assert.Equal(t, "/ui", sessionCookie.Path)
}

func TestAuthHandler_BackchannelLogout_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var calledToken string
	mockUC := &mockSessionUseCase{
		BackchannelLogoutFunc: func(ctx context.Context, rawLogoutToken string) error {
			calledToken = rawLogoutToken
			return nil
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Set("logout_token", "mock-logout-token-123")
	req, _ := http.NewRequest(http.MethodPost, "/ui/api/v1/auth/backchannel-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "mock-logout-token-123", calledToken)
}

func TestAuthHandler_BackchannelLogout_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		BackchannelLogoutFunc: func(ctx context.Context, rawLogoutToken string) error {
			return errors.New("invalid token")
		},
	}

	bffhttp.RegisterRoutes(r, mockUC, nil, "/ui")

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Set("logout_token", "invalid-token")
	req, _ := http.NewRequest(http.MethodPost, "/ui/api/v1/auth/backchannel-logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token", resp["error"])
}
