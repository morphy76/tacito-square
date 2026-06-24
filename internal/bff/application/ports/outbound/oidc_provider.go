package outbound

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

// TokenSet holds the raw token credentials returned by the OIDC identity provider.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresIn    time.Duration
}

// OIDCProvider defines the outbound port for interacting with the OIDC Identity Provider (OP).
type OIDCProvider interface {
	// ExchangeCode exchanges an authorization code for a TokenSet.
	ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenSet, error)

	// RefreshToken uses a refresh token to obtain a fresh TokenSet.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error)

	// FetchUserInfo retrieves the OIDC UserInfo payload using the provided access token.
	FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error)

	// ValidateLogoutToken parses and validates a backchannel logout token.
	// Returns the subject (sub) and OIDC session ID (sid) claims if valid.
	ValidateLogoutToken(ctx context.Context, rawToken string) (sub string, sessionID string, err error)

	// ValidateAccessToken validates an Access Token (JWT) statelessly and returns its claims.
	ValidateAccessToken(ctx context.Context, token string) (*model.UserInfoPayload, error)
}
