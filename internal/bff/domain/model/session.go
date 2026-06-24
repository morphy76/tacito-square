package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session is the BFF domain aggregate that represents an authenticated user session.
// It holds the OIDC tokens and resolved tenant identity for a single browser session.
// Tokens are tagged json:"-" to prevent accidental serialization in log output or API
// responses; the Redis adapter uses a dedicated serialization strategy.
type Session struct {
	// ID is a cryptographically random UUID v4 that identifies the BFF session.
	// It is bound to the browser via an HttpOnly, Secure cookie.
	ID string `json:"id"`

	// UserID is the OIDC subject claim (sub) identifying the authenticated user.
	// Used for sub-based backchannel logout: all BFF sessions for this user are
	// invalidated when the OP sends a Logout Token carrying only the sub claim.
	UserID string `json:"user_id"`

	// TenantID is the resolved tenant identifier, extracted from the UserInfo payload.
	TenantID string `json:"tenant_id"`

	// Issuer is the iss claim of the OIDC provider that created the tokens for this
	// session (e.g., "https://keycloak.example.com/realms/tacito").
	// Stored to validate that an inbound backchannel Logout Token originates from
	// the same OP that authenticated the user, preventing cross-issuer attacks.
	Issuer string `json:"issuer"`

	// OIDCSessionID is the sid claim from the OIDC ID Token.
	// When the OP sends a Logout Token containing a sid claim (with or without sub),
	// the BFF uses this field to locate and invalidate the exact matching BFF session.
	// May be empty if the OP did not issue a session ID in the ID Token.
	OIDCSessionID string `json:"oidc_session_id"`

	// AccessToken is the opaque OIDC access token. Never serialized to JSON.
	AccessToken string `json:"-"`

	// RefreshToken is the OIDC refresh token used for transparent token renewal.
	// Never serialized to JSON.
	RefreshToken string `json:"-"`

	// IDToken is the raw OIDC ID Token JWT. Never serialized to JSON.
	IDToken string `json:"-"`

	// UserInfo is the cached OIDC UserInfo payload associated with this session.
	UserInfo UserInfoPayload `json:"user_info"`

	// AccessTokenExpiresAt is the UTC timestamp at which the access token expires.
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`

	// CreatedAt is the UTC timestamp at which the session was created.
	CreatedAt time.Time `json:"created_at"`

	// Invalidated is set to true when the session is explicitly destroyed
	// (logout or backchannel logout). An invalidated session must not be refreshed.
	Invalidated bool `json:"invalidated"`
}

// NewSession constructs a new Session aggregate with a cryptographically random
// UUID v4 session ID. The accessTokenTTL parameter drives AccessTokenExpiresAt.
// issuer is the iss claim of the OIDC provider; oidcSessionID is the optional
// sid claim from the ID Token (pass an empty string if the OP did not issue one).
func NewSession(
	userID string,
	tenantID string,
	issuer string,
	oidcSessionID string,
	accessToken string,
	refreshToken string,
	idToken string,
	userInfo UserInfoPayload,
	accessTokenTTL time.Duration,
) (*Session, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generating session ID: %w", err)
	}

	now := time.Now().UTC()
	return &Session{
		ID:                   id.String(),
		UserID:               userID,
		TenantID:             tenantID,
		Issuer:               issuer,
		OIDCSessionID:        oidcSessionID,
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		IDToken:              idToken,
		UserInfo:             userInfo,
		AccessTokenExpiresAt: now.Add(accessTokenTTL),
		CreatedAt:            now,
		Invalidated:          false,
	}, nil
}

// IsExpired reports whether the session's access token has passed its expiry time.
// Callers should attempt a transparent token refresh when this returns true.
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.AccessTokenExpiresAt)
}

// Invalidate clears all sensitive token material and marks the session as invalidated.
// After invalidation the session must be removed from the store and must not be refreshed.
func (s *Session) Invalidate() {
	s.AccessToken = ""
	s.RefreshToken = ""
	s.Invalidated = true
}
