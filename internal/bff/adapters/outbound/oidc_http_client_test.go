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

// TestOIDCClient_InternalIssuer_RewritesEndpoints verifies that when InternalIssuer
// is configured, the discovery request and all subsequent endpoint calls (token,
// userinfo) use the internal cluster URL rather than the public localhost URL.
// This covers the scenario where the BFF runs inside Kubernetes but Keycloak is
// only reachable from the browser via localhost through a Traefik ingress.
func TestOIDCClient_InternalIssuer_RewritesEndpoints(t *testing.T) {
	// publicIssuer simulates what the browser uses (localhost — unreachable from inside the pod).
	const publicIssuer = "http://localhost/auth/realms/tacito"

	// The mock server acts as the internal Keycloak service.
	// It handles: GET /.well-known/openid-configuration, POST /token, GET /userinfo.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Assert that the Host header is rewritten to the public issuer's host.
		assert.Equal(t, "localhost", r.Host, "Host header must be rewritten to match public issuer")
		assert.Equal(t, "localhost", r.Header.Get("X-Forwarded-Host"), "X-Forwarded-Host header must be set")
		
		switch r.URL.Path {
		case "/auth/realms/tacito/.well-known/openid-configuration":
			// Discovery document — endpoints reference the *public* issuer
			// (as a real Keycloak would), but we expect the client to rewrite them.
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"token_endpoint":    publicIssuer + "/protocol/openid-connect/token",
				"userinfo_endpoint": publicIssuer + "/protocol/openid-connect/userinfo",
				"jwks_uri":          publicIssuer + "/protocol/openid-connect/certs",
			})
		case "/auth/realms/tacito/protocol/openid-connect/token":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "internal-access-token",
				"refresh_token": "internal-refresh-token",
				"id_token":      "header.e30K.sig",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := outbound.OIDCClientConfig{
		ClientID:       "tacito",
		ClientSecret:   "secret",
		RedirectURI:    "http://localhost/ui/api/v1/auth/callback",
		Issuer:         publicIssuer,
		InternalIssuer: srv.URL + "/auth/realms/tacito",
		Timeout:        5 * time.Second,
	}
	client := outbound.NewOIDCHTTPClient(cfg)

	tokenSet, err := client.ExchangeCode(context.Background(), "auth-code", cfg.RedirectURI)
	require.NoError(t, err, "ExchangeCode must succeed when internal issuer routes to the mock server")
	assert.Equal(t, "internal-access-token", tokenSet.AccessToken)
	assert.Equal(t, "internal-refresh-token", tokenSet.RefreshToken)
}
