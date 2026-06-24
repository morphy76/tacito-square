package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	bffhttp "github.com/morphy76/tacito-square/internal/bff/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

func TestSessionMiddleware_ValidCookie_PopulatesContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			assert.Equal(t, "session-123", sessionID)
			return &model.Session{
				ID:       "session-123",
				UserID:   "user-456",
				TenantID: "tenant-789",
			}, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		tenantID, _ := c.Get("tenantID")
		userID, _ := c.Get("userID")
		assert.Equal(t, "tenant-789", tenantID)
		assert.Equal(t, "user-456", userID)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSessionMiddleware_MissingCookie_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSessionMiddleware_ExpiredSession_AttemptRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	getCalls := 0
	refreshCalls := 0

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			getCalls++
			return nil, model.ErrSessionExpired
		},
		RefreshSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			refreshCalls++
			assert.Equal(t, "session-123", sessionID)
			return &model.Session{
				ID:       "new-session-456",
				UserID:   "user-456",
				TenantID: "tenant-789",
			}, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		tenantID, _ := c.Get("tenantID")
		assert.Equal(t, "tenant-789", tenantID)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, refreshCalls)

	// Check that a new cookie was set in the response
	cookies := w.Result().Cookies()
	var newCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "bff_session_id" {
			newCookie = c
		}
	}
	assert.NotNil(t, newCookie)
	assert.Equal(t, "new-session-456", newCookie.Value)
	assert.Equal(t, "/ui", newCookie.Path)
}

func TestSessionMiddleware_RefreshFails_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return nil, model.ErrSessionExpired
		},
		RefreshSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return nil, model.ErrSessionNotFound
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Session cookie should be cleared (expired) in the response
	cookies := w.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "bff_session_id" {
			clearedCookie = c
		}
	}
	assert.NotNil(t, clearedCookie)
	assert.True(t, clearedCookie.Expires.Before(time.Now()))
	assert.Equal(t, "/ui", clearedCookie.Path)
}

func TestAuthRedirectMiddleware_ValidCookie_Continues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			assert.Equal(t, "session-123", sessionID)
			return &model.Session{
				ID:       "session-123",
				UserID:   "user-456",
				TenantID: "tenant-789",
			}, nil
		},
	}

	r.Use(bffhttp.AuthRedirectMiddleware(mockUC, "/ui"))
	r.GET("/test-secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-secure", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRedirectMiddleware_MissingCookie_Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{}

	r.Use(bffhttp.AuthRedirectMiddleware(mockUC, "/ui"))
	r.GET("/test-secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-secure", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui/api/v1/auth/login?redirect_to=%2Ftest-secure", w.Header().Get("Location"))
}

func TestAuthRedirectMiddleware_ExpiredSession_Redirects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return nil, model.ErrSessionExpired
		},
		RefreshSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return nil, model.ErrSessionNotFound
		},
	}

	r.Use(bffhttp.AuthRedirectMiddleware(mockUC, "/ui"))
	r.GET("/test-secure", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-secure", nil)
	req.AddCookie(&http.Cookie{
		Name:  "bff_session_id",
		Value: "session-123",
	})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui/api/v1/auth/login?redirect_to=%2Ftest-secure", w.Header().Get("Location"))
}

func TestAuthRedirectMiddleware_AppendsRedirectToQueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{}

	r.Use(bffhttp.AuthRedirectMiddleware(mockUC, "/ui"))
	r.GET("/ui/secure/dashboard", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ui/secure/dashboard", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(t, location, "/ui/api/v1/auth/login")
	assert.Contains(t, location, "redirect_to=")
	assert.Contains(t, location, "%2Fui%2Fsecure%2Fdashboard")
}

func TestSessionMiddleware_ValidBearerToken_Succeeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	getSessionCalled := false
	mockUC := &mockSessionUseCase{
		ValidateAccessTokenFunc: func(ctx context.Context, token string) (*model.UserInfoPayload, error) {
			assert.Equal(t, "valid-token-123", token)
			return &model.UserInfoPayload{
				Sub:      "user-jwt",
				TenantID: "tenant-jwt",
				Email:    "jwt@test.com",
			}, nil
		},
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			getSessionCalled = true
			return nil, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		tenantID, _ := c.Get("tenantID")
		userID, _ := c.Get("userID")
		assert.Equal(t, "tenant-jwt", tenantID)
		assert.Equal(t, "user-jwt", userID)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token-123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, getSessionCalled, "Session retrieval should NOT be called when Bearer token is present")
}

func TestSessionMiddleware_ExpiredBearerToken_Returns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		ValidateAccessTokenFunc: func(ctx context.Context, token string) (*model.UserInfoPayload, error) {
			return nil, errors.New("token is expired")
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer expired-token-123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}


