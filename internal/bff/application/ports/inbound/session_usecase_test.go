package inbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

type mockSessionUseCase struct{}

func (m *mockSessionUseCase) InitiateLogin(ctx context.Context, redirectTo string) (authURL string, state string, err error) {
	return "", "", nil
}

func (m *mockSessionUseCase) HandleCallback(ctx context.Context, code, state string) (*model.Session, string, error) {
	return nil, "", nil
}

func (m *mockSessionUseCase) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return nil, nil
}

func (m *mockSessionUseCase) Logout(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionUseCase) BackchannelLogout(ctx context.Context, rawLogoutToken string) error {
	return nil
}

func (m *mockSessionUseCase) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return nil, nil
}

// Compile-time assertion
var _ inbound.SessionUseCase = (*mockSessionUseCase)(nil)

func TestSessionUseCaseInterface(t *testing.T) {
	// Assertions are checked at compile time
}
