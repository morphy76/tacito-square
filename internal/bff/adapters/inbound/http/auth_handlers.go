package http

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
)

type AuthHandler struct {
	sessionUC inbound.SessionUseCase
	uiPath    string
}

func NewAuthHandler(sessionUC inbound.SessionUseCase, uiPath string) *AuthHandler {
	return &AuthHandler{
		sessionUC: sessionUC,
		uiPath:    uiPath,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	// Capture the original URL the user was trying to access so we can redirect
	// back to it after a successful login.
	redirectTo := c.Query("redirect_to")

	authURL, state, err := h.sessionUC.InitiateLogin(c.Request.Context(), redirectTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stateCookie := &http.Cookie{
		Name:     "bff_oidc_state",
		Value:    state,
		Path:     h.uiPath,
		Domain:   "",
		Expires:  time.Now().Add(5 * time.Minute),
		MaxAge:   300,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, stateCookie)

	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	stateCookie, err := c.Request.Cookie("bff_oidc_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	// Clear the state cookie
	clearedStateCookie := &http.Cookie{
		Name:     "bff_oidc_state",
		Value:    "",
		Path:     h.uiPath,
		Domain:   "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, clearedStateCookie)

	sess, redirectTo, err := h.sessionUC.HandleCallback(c.Request.Context(), code, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write session cookie
	sessionCookie := &http.Cookie{
		Name:     "bff_session_id",
		Value:    sess.ID,
		Path:     h.uiPath,
		Domain:   "",
		Expires:  time.Now().Add(365 * 24 * time.Hour), // long-lived, server manages lifecycle
		MaxAge:   0,                                    // session-scoped
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, sessionCookie)

	// Redirect to the originally-requested resource if stored, otherwise fall back to uiPath.
	target := redirectTo
	if target == "" {
		target = h.uiPath
	}
	c.Redirect(http.StatusFound, target)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	sessionCookie, err := c.Request.Cookie("bff_session_id")
	if err != nil || sessionCookie.Value == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sessionID := sessionCookie.Value

	// Try to get session to extract issuer
	var issuer string
	sess, err := h.sessionUC.GetSession(c.Request.Context(), sessionID)
	if err == nil && sess != nil {
		issuer = sess.Issuer
	}

	// Call use case to invalidate session
	_ = h.sessionUC.Logout(c.Request.Context(), sessionID)

	// Clear session cookie
	clearedCookie := &http.Cookie{
		Name:     "bff_session_id",
		Value:    "",
		Path:     h.uiPath,
		Domain:   "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, clearedCookie)

	// Redirect to OIDC end-session endpoint
	redirectURL := h.uiPath
	if issuer != "" {
		u, err := url.Parse(issuer)
		if err == nil {
			if !strings.HasSuffix(u.Path, "/protocol/openid-connect/logout") {
				u.Path = strings.TrimSuffix(u.Path, "/") + "/protocol/openid-connect/logout"
			}
			redirectURL = u.String()
		}
	}

	c.Redirect(http.StatusFound, redirectURL)
}

func (h *AuthHandler) BackchannelLogout(c *gin.Context) {
	logoutToken := c.PostForm("logout_token")
	if logoutToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing logout token"})
		return
	}

	err := h.sessionUC.BackchannelLogout(c.Request.Context(), logoutToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
