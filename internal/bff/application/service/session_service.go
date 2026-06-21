package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	userIDKey   contextKey = "user_id"
)

// ContextWithTenantID returns a derived context carrying the resolved tenant ID.
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// TenantIDFromContext extracts the tenant ID from the context.
func TenantIDFromContext(ctx context.Context) string {
	val, _ := ctx.Value(tenantIDKey).(string)
	return val
}

// ContextWithUserID returns a derived context carrying the resolved user ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	val, _ := ctx.Value(userIDKey).(string)
	return val
}

// SessionConfig contains OIDC provider client configuration parameters.
type SessionConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	SessionTTL   time.Duration
	Issuer       string
}

// SessionService coordinates user authentication and session token lifecycles.
type SessionService struct {
	store        outbound.SessionStore
	oidc         outbound.OIDCProvider
	cfg          SessionConfig
	refreshGroup singleflight.Group
}

// NewSessionService constructs a new SessionService.
func NewSessionService(store outbound.SessionStore, oidc outbound.OIDCProvider, cfg SessionConfig) *SessionService {
	return &SessionService{
		store: store,
		oidc:  oidc,
		cfg:   cfg,
	}
}

// InitiateLogin generates the OIDC authorization URL and state token.
func (s *SessionService) InitiateLogin(ctx context.Context) (string, string, error) {
	state := uuid.New().String()

	u, err := url.Parse(s.cfg.Issuer)
	if err != nil {
		return "", "", fmt.Errorf("invalid issuer URL: %w", err)
	}

	if !strings.HasSuffix(u.Path, "/protocol/openid-connect/auth") {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/protocol/openid-connect/auth"
	}

	q := u.Query()
	q.Set("client_id", s.cfg.ClientID)
	q.Set("redirect_uri", s.cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	u.RawQuery = q.Encode()

	return u.String(), state, nil
}

// HandleCallback exchanges an auth code for tokens, retrieves user claims, and creates a session.
func (s *SessionService) HandleCallback(ctx context.Context, code, state string) (*model.Session, error) {
	tokens, err := s.oidc.ExchangeCode(ctx, code, s.cfg.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	userInfo, err := s.oidc.FetchUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	oidcSessionID := extractSessionID(tokens.IDToken)

	session, err := model.NewSession(
		userInfo.Sub,
		userInfo.TenantID,
		s.cfg.Issuer,
		oidcSessionID,
		tokens.AccessToken,
		tokens.RefreshToken,
		*userInfo,
		tokens.ExpiresIn,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session aggregate: %w", err)
	}

	if err := s.store.Save(ctx, session, s.cfg.SessionTTL); err != nil {
		return nil, fmt.Errorf("failed to persist session: %w", err)
	}

	log.Info().
		Str("session_id", session.ID).
		Str("user_id", session.UserID).
		Str("tenant_id", session.TenantID).
		Msg("BFF session created")

	return session, nil
}

// RefreshSession performs a thundering-herd protected OIDC token refresh if expired.
func (s *SessionService) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	val, err, _ := s.refreshGroup.Do(sessionID, func() (interface{}, error) {
		session, err := s.store.Get(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		if !session.IsExpired() {
			return session, nil
		}

		if session.Invalidated {
			return nil, model.ErrSessionInvalidated
		}

		tokens, err := s.oidc.RefreshToken(ctx, session.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh tokens with OIDC provider: %w", err)
		}

		session.AccessToken = tokens.AccessToken
		session.RefreshToken = tokens.RefreshToken
		session.AccessTokenExpiresAt = time.Now().Add(tokens.ExpiresIn)

		if err := s.store.Save(ctx, session, s.cfg.SessionTTL); err != nil {
			return nil, fmt.Errorf("failed to save refreshed session: %w", err)
		}

		log.Info().
			Str("session_id", session.ID).
			Str("user_id", session.UserID).
			Msg("BFF session tokens refreshed")

		return session, nil
	})

	if err != nil {
		return nil, err
	}
	return val.(*model.Session), nil
}

// Logout invalidates a session locally by deleting it from the session store.
func (s *SessionService) Logout(ctx context.Context, sessionID string) error {
	if err := s.store.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	log.Info().
		Str("session_id", sessionID).
		Msg("BFF session invalidated/logged out")

	return nil
}

// BackchannelLogout processes an incoming backchannel logout request from the OIDC Provider.
func (s *SessionService) BackchannelLogout(ctx context.Context, rawLogoutToken string) error {
	sub, sid, err := s.oidc.ValidateLogoutToken(ctx, rawLogoutToken)
	if err != nil {
		return fmt.Errorf("failed to validate logout token: %w", err)
	}

	if sid != "" {
		if err := s.store.DeleteByOIDCSessionID(ctx, s.cfg.Issuer, sid); err != nil {
			return fmt.Errorf("failed to delete sessions by OIDC session ID: %w", err)
		}
		log.Info().
			Str("sid", sid).
			Msg("Backchannel logout processed by OIDC session ID")
		return nil
	}

	if sub != "" {
		if err := s.store.DeleteByUserID(ctx, sub); err != nil {
			return fmt.Errorf("failed to delete sessions by user ID: %w", err)
		}
		log.Info().
			Str("sub", sub).
			Msg("Backchannel logout processed by user ID")
		return nil
	}

	return errors.New("invalid logout token: neither sub nor sid claim present")
}

// GetSession retrieves the session state if active and not expired.
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.IsExpired() {
		return nil, model.ErrSessionExpired
	}
	if session.Invalidated {
		return nil, model.ErrSessionInvalidated
	}
	return session, nil
}

// extractSessionID extracts the OIDC sid claim from an ID token JWT payload without verifying its signature.
func extractSessionID(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Sid string `json:"sid"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Sid
}
