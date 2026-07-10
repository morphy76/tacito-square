package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// SessionMiddleware authenticates incoming requests using a session cookie or stateless Bearer token.
func SessionMiddleware(sessionUC inbound.SessionUseCase, uiPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 1. Check Bearer Token first
		if token := getBearerToken(c.Request); token != "" {
			userInfo, err := sessionUC.ValidateAccessToken(ctx, token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
			enrichContext(c, userInfo)
			c.Next()
			return
		}

		// 2. Fallback to Cookie
		sessionCookie, err := c.Request.Cookie("bff_session_id")
		if err != nil || sessionCookie.Value == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		sessionID := sessionCookie.Value
		sess, err := sessionUC.GetSession(ctx, sessionID)
		if err != nil || sess == nil {
			if err != nil && errors.Is(err, model.ErrSessionExpired) {
				// Attempt to refresh the session
				newSess, rerr := sessionUC.RefreshSession(ctx, sessionID)
				if rerr != nil {
					// Refresh failed: clear cookie and abort
					clearSessionCookie(c, uiPath)
					c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					c.Abort()
					return
				}

				// Success: set new session cookie
				setSessionCookie(c, newSess.ID, uiPath)
				sess = newSess
			} else {
				// Other error (e.g. SessionNotFound, SessionInvalidated or sess == nil)
				clearSessionCookie(c, uiPath)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				c.Abort()
				return
			}
		}

		userInfo := sess.UserInfo
		if userInfo.Sub == "" {
			userInfo.Sub = sess.UserID
		}
		if userInfo.TenantID == "" {
			userInfo.TenantID = sess.TenantID
		}

		enrichContext(c, &userInfo)
		c.Next()
	}
}

func setSessionCookie(c *gin.Context, sessionID string, uiPath string) {
	cookie := &http.Cookie{
		Name:     "bff_session_id",
		Value:    sessionID,
		Path:     uiPath,
		Domain:   "",
		Expires:  time.Now().Add(365 * 24 * time.Hour), // long lived, but actual session validation is server-side/redis
		MaxAge:   0,                                    // session-scoped
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

func clearSessionCookie(c *gin.Context, uiPath string) {
	cookie := &http.Cookie{
		Name:     "bff_session_id",
		Value:    "",
		Path:     uiPath,
		Domain:   "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

// AuthRedirectMiddleware redirects unauthenticated users to the login endpoint, appending the
// original request URI as a redirect_to query parameter so the Login handler can restore it
// after a successful login. It also supports stateless Bearer tokens.
func AuthRedirectMiddleware(sessionUC inbound.SessionUseCase, uiPath string) gin.HandlerFunc {
	loginBase := strings.TrimSuffix(uiPath, "/") + "/api/v1/auth/login"

	buildLoginURL := func(c *gin.Context) string {
		requestURI := c.Request.RequestURI
		if requestURI == "" {
			requestURI = c.Request.URL.RequestURI()
		}
		if requestURI == "" || requestURI == "/" {
			return loginBase
		}
		return loginBase + "?redirect_to=" + url.QueryEscape(requestURI)
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 1. Check Bearer Token first
		if token := getBearerToken(c.Request); token != "" {
			userInfo, err := sessionUC.ValidateAccessToken(ctx, token)
			if err != nil {
				c.Redirect(http.StatusFound, buildLoginURL(c))
				c.Abort()
				return
			}
			enrichContext(c, userInfo)
			c.Next()
			return
		}

		// 2. Fallback to Cookie
		sessionCookie, err := c.Request.Cookie("bff_session_id")
		if err != nil || sessionCookie.Value == "" {
			c.Redirect(http.StatusFound, buildLoginURL(c))
			c.Abort()
			return
		}

		sessionID := sessionCookie.Value
		sess, err := sessionUC.GetSession(ctx, sessionID)
		if err != nil || sess == nil {
			if err != nil && errors.Is(err, model.ErrSessionExpired) {
				// Attempt to refresh the session
				newSess, rerr := sessionUC.RefreshSession(ctx, sessionID)
				if rerr != nil {
					clearSessionCookie(c, uiPath)
					c.Redirect(http.StatusFound, buildLoginURL(c))
					c.Abort()
					return
				}
				setSessionCookie(c, newSess.ID, uiPath)
				sess = newSess
			} else {
				// Other error (e.g. SessionNotFound, SessionInvalidated or sess == nil)
				clearSessionCookie(c, uiPath)
				c.Redirect(http.StatusFound, buildLoginURL(c))
				c.Abort()
				return
			}
		}

		userInfo := sess.UserInfo
		if userInfo.Sub == "" {
			userInfo.Sub = sess.UserID
		}
		if userInfo.TenantID == "" {
			userInfo.TenantID = sess.TenantID
		}

		enrichContext(c, &userInfo)
		c.Next()
	}
}

func getBearerToken(req *http.Request) string {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func enrichContext(c *gin.Context, userInfo *model.UserInfoPayload) {
	c.Set("userInfo", userInfo)
	c.Set("tenantID", userInfo.TenantID)
	c.Set("userID", userInfo.Sub)

	ctx := c.Request.Context()

	// Parse and build tenant context
	if userInfo.TenantID != "" {
		t, err := tenant.New(userInfo.TenantID, userInfo.SubscriptionID)
		if err == nil {
			ctx = tenant.ContextWithTenant(ctx, t)
		}
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetAttributes(
			attribute.String("tenant_id", userInfo.TenantID),
			attribute.String("user_id", userInfo.Sub),
		)
	}

	logger := zerolog.Ctx(ctx).With().Str("tenant_id", userInfo.TenantID).Logger()
	c.Request = c.Request.WithContext(logger.WithContext(ctx))
}

// RequireRoles checks if the authenticated user has at least one of the allowed roles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("userInfo")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		userInfo, ok := val.(*model.UserInfoPayload)
		if !ok || userInfo == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		for _, r := range userInfo.Roles {
			for _, allowed := range allowedRoles {
				if r == allowed {
					c.Next()
					return
				}
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}
