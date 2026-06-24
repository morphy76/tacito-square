package inbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

// SessionUseCase defines the inbound port (driving interface) for session lifecycle orchestrations.
type SessionUseCase interface {
	// InitiateLogin generates the OIDC authorization URL and state token, storing the redirectTo
	// URL so it can be recovered after the callback.
	InitiateLogin(ctx context.Context, redirectTo string) (authURL string, state string, err error)

	// HandleCallback handles the OIDC redirect callback, exchanging the code and creating a session.
	// It also returns the redirectTo URL that was stored during InitiateLogin, or an empty string if none.
	HandleCallback(ctx context.Context, code, state string) (*model.Session, string, error)

	// RefreshSession transparently refreshes tokens for an active session.
	RefreshSession(ctx context.Context, sessionID string) (*model.Session, error)

	// Logout invalidates a session locally.
	Logout(ctx context.Context, sessionID string) error

	// BackchannelLogout processes an incoming backchannel logout request from the OIDC Provider.
	BackchannelLogout(ctx context.Context, rawLogoutToken string) error

	// GetSession retrieves the session state if active.
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)

	// ValidateAccessToken validates an Access Token (JWT) statelessly and returns its claims.
	ValidateAccessToken(ctx context.Context, token string) (*model.UserInfoPayload, error)
}
