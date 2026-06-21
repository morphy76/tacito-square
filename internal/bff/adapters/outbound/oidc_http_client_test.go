package outbound_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/adapters/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCClient_ExchangeCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access-token",
			"refresh_token": "test-refresh-token",
			"id_token":      "header.e30K.sig",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	cfg := outbound.OIDCClientConfig{
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		RedirectURI:   "http://localhost/callback",
		TokenEndpoint: srv.URL + "/token",
		Timeout:       5 * time.Second,
	}
	client := outbound.NewOIDCHTTPClient(cfg)

	tokenSet, err := client.ExchangeCode(context.Background(), "auth-code", cfg.RedirectURI)
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", tokenSet.AccessToken)
	assert.Equal(t, "test-refresh-token", tokenSet.RefreshToken)
	assert.Equal(t, 1*time.Hour, tokenSet.ExpiresIn)
}

func TestOIDCClient_FetchUserInfo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":            "user-sub-123",
			"email":          "user@tacito.local",
			"tenantid":       "tenant-a",
			"subscriptionid": "sub-1",
		})
	}))
	defer srv.Close()

	cfg := outbound.OIDCClientConfig{
		UserInfoEndpoint: srv.URL + "/userinfo",
		Timeout:          5 * time.Second,
	}
	client := outbound.NewOIDCHTTPClient(cfg)

	userInfo, err := client.FetchUserInfo(context.Background(), "access-token")
	require.NoError(t, err)
	assert.Equal(t, "user-sub-123", userInfo.Sub)
	assert.Equal(t, "tenant-a", userInfo.TenantID)
}

func TestOIDCClient_CircuitBreaker(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	const maxFail = 3
	cfg := outbound.OIDCClientConfig{
		ClientID:              "client-id",
		ClientSecret:          "secret",
		RedirectURI:           "http://localhost/callback",
		TokenEndpoint:         srv.URL + "/token",
		Timeout:               1 * time.Second,
		CircuitBreakerMaxFail: maxFail,
	}
	client := outbound.NewOIDCHTTPClient(cfg)

	// Exhaust the circuit breaker threshold
	for i := 0; i < maxFail; i++ {
		_, _ = client.ExchangeCode(context.Background(), "code", cfg.RedirectURI)
	}

	callsBefore := callCount
	// Next call must be short-circuited (circuit is open)
	_, err := client.ExchangeCode(context.Background(), "code", cfg.RedirectURI)
	assert.Error(t, err)
	assert.Equal(t, callsBefore, callCount, "circuit breaker should prevent further calls to server")
}
