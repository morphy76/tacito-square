package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextWithClaims_RoundTrip(t *testing.T) {
	claims := &Claims{
		Subject: "user-123",
		Email:   "user@example.com",
		Roles:   []string{"admin", "user"},
	}

	ctx := ContextWithClaims(context.Background(), claims)
	got := ClaimsFromContext(ctx)

	require.NotNil(t, got)
	assert.Equal(t, "user-123", got.Subject)
	assert.Equal(t, "user@example.com", got.Email)
	assert.Equal(t, []string{"admin", "user"}, got.Roles)
}

func TestClaimsFromContext_ReturnsNilWhenMissing(t *testing.T) {
	got := ClaimsFromContext(context.Background())
	assert.Nil(t, got)
}

func TestExtractBearerToken_ValidHeader(t *testing.T) {
	token, err := ExtractBearerToken("Bearer eyJhbGciOiJIUzI1NiJ9.test")
	require.NoError(t, err)
	assert.Equal(t, "eyJhbGciOiJIUzI1NiJ9.test", token)
}

func TestExtractBearerToken_CaseInsensitive(t *testing.T) {
	token, err := ExtractBearerToken("bearer my-token")
	require.NoError(t, err)
	assert.Equal(t, "my-token", token)
}

func TestExtractBearerToken_MissingHeader(t *testing.T) {
	_, err := ExtractBearerToken("")
	assert.Error(t, err)
}

func TestExtractBearerToken_InvalidFormat(t *testing.T) {
	_, err := ExtractBearerToken("Basic dXNlcjpwYXNz")
	assert.Error(t, err)
}

func TestExtractBearerToken_EmptyToken(t *testing.T) {
	_, err := ExtractBearerToken("Bearer  ")
	assert.Error(t, err)
}
