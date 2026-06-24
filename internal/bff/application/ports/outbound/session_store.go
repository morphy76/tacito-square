package outbound

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

// SessionStore defines the outbound port for managing browser sessions in persistent storage.
type SessionStore interface {
	// Save stores the session with a configured Time-To-Live.
	Save(ctx context.Context, session *model.Session, ttl time.Duration) error

	// Get retrieves a session by its unique ID. Returns ErrSessionNotFound if not found.
	Get(ctx context.Context, sessionID string) (*model.Session, error)

	// Delete removes a specific session by its ID.
	Delete(ctx context.Context, sessionID string) error

	// DeleteByUserID removes all sessions associated with a specific user ID (used for sub-based backchannel logout).
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteByOIDCSessionID removes a specific session associated with the given issuer and OIDC sid claim.
	DeleteByOIDCSessionID(ctx context.Context, issuer string, oidcSessionID string) error

	// SavePendingState stores the redirect-to URL keyed by the OIDC state nonce.
	// The entry should expire after a short TTL (e.g. 5 minutes) matching the state cookie lifetime.
	SavePendingState(ctx context.Context, state, redirectTo string, ttl time.Duration) error

	// GetAndDeletePendingState atomically retrieves and removes the redirect-to URL for the given
	// state nonce. Returns an empty string (no error) when the key is not found.
	GetAndDeletePendingState(ctx context.Context, state string) (redirectTo string, err error)
}
