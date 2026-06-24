package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
)

type mockOIDCProvider struct{}

func (m *mockOIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) {
	return nil, nil
}

func (m *mockOIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) {
	return nil, nil
}

func (m *mockOIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) {
	return nil, nil
}

func (m *mockOIDCProvider) ValidateLogoutToken(ctx context.Context, rawToken string) (sub string, sessionID string, err error) {
	return "", "", nil
}

func (m *mockOIDCProvider) ValidateAccessToken(ctx context.Context, token string) (*model.UserInfoPayload, error) {
	return nil, nil
}

// Compile-time assertion
var _ outbound.OIDCProvider = (*mockOIDCProvider)(nil)

func TestOIDCProviderInterface(t *testing.T) {
	// Assertions are checked at compile time
}
