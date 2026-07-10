package outbound_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

type mockSessionStore struct{}

func (m *mockSessionStore) Save(ctx context.Context, session *model.Session, ttl time.Duration) error {
	return nil
}

func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	return nil, nil
}

func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	return nil
}

func (m *mockSessionStore) DeleteByOIDCSessionID(ctx context.Context, issuer string, oidcSessionID string) error {
	return nil
}

func (m *mockSessionStore) SavePendingState(ctx context.Context, state, redirectTo string, ttl time.Duration) error {
	return nil
}

func (m *mockSessionStore) GetAndDeletePendingState(ctx context.Context, state string) (string, error) {
	return "", nil
}

func (m *mockSessionStore) CacheHTML(ctx context.Context, key string, html string, ttl time.Duration) error {
	return nil
}

func (m *mockSessionStore) GetCachedHTML(ctx context.Context, key string) (string, error) {
	return "", nil
}

// Compile-time assertion
var _ outbound.SessionStore = (*mockSessionStore)(nil)

func TestSessionStoreInterface(t *testing.T) {
	// Assertions are checked at compile time
}
