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
)

// SessionMiddleware authenticates incoming requests using a session cookie.
func SessionMiddleware(sessionUC inbound.SessionUseCase, uiPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Request.Cookie("bff_session_id")
		if err != nil || sessionCookie.Value == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		sessionID := sessionCookie.Value
		ctx := c.Request.Context()

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

		// Enrich Gin context
		c.Set("tenantID", sess.TenantID)
		c.Set("userID", sess.UserID)

		// Enrich OTel span attributes
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.String("tenant_id", sess.TenantID),
				attribute.String("user_id", sess.UserID),
			)
		}

		// Inject tenantID into zerolog logger context
		logger := zerolog.Ctx(ctx).With().Str("tenant_id", sess.TenantID).Logger()
		c.Request = c.Request.WithContext(logger.WithContext(ctx))

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
// after a successful login.
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
		sessionCookie, err := c.Request.Cookie("bff_session_id")
		if err != nil || sessionCookie.Value == "" {
			c.Redirect(http.StatusFound, buildLoginURL(c))
			c.Abort()
			return
		}

		sessionID := sessionCookie.Value
		ctx := c.Request.Context()

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

		// Enrich Gin context
		c.Set("tenantID", sess.TenantID)
		c.Set("userID", sess.UserID)

		// Enrich OTel span attributes
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.String("tenant_id", sess.TenantID),
				attribute.String("user_id", sess.UserID),
			)
		}

		// Inject tenantID into zerolog logger context
		logger := zerolog.Ctx(ctx).With().Str("tenant_id", sess.TenantID).Logger()
		c.Request = c.Request.WithContext(logger.WithContext(ctx))

		c.Next()
	}
}
