package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	gobreaker "github.com/sony/gobreaker/v2"
	zoidc "github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/client/rp"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/rs/zerolog/log"
)

// Compile-time interface satisfaction assertion.
var _ outbound.OIDCProvider = (*OIDCHTTPClient)(nil)

// OIDCClientConfig holds configurable parameters for the OIDC outbound adapter.
type OIDCClientConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Issuer       string

	// InternalIssuer is an optional override for the OIDC discovery and endpoint calls.
	// Set this when the BFF runs inside a Kubernetes cluster and the public Issuer URL
	// (e.g. http://localhost/auth/realms/tacito exposed via a Traefik ingress) is not
	// reachable from within the pod. The BFF will perform discovery against InternalIssuer
	// and rewrite any discovered endpoints that reference the public Issuer to use
	// InternalIssuer instead. Token validation still checks the public Issuer claim.
	InternalIssuer string

	// Explicit endpoint overrides (used in tests; in production populated from OIDC discovery).
	TokenEndpoint    string
	UserInfoEndpoint string

	Timeout               time.Duration
	CircuitBreakerMaxFail uint32
}

// OIDCHTTPClient is a driven adapter that implements outbound.OIDCProvider
// using the github.com/zitadel/oidc/v3 Relying Party helpers.
type OIDCHTTPClient struct {
	cfg     OIDCClientConfig
	httpCli *http.Client
	cb      *gobreaker.CircuitBreaker[[]byte]

	mu               sync.Mutex
	discovered       bool
	tokenEndpoint    string
	userinfoEndpoint string
	jwksURI          string
}

// NewOIDCHTTPClient constructs an OIDCHTTPClient with a circuit breaker and a configured HTTP client.
func NewOIDCHTTPClient(cfg OIDCClientConfig) *OIDCHTTPClient {
	maxFail := cfg.CircuitBreakerMaxFail
	if maxFail == 0 {
		maxFail = 5
	}

	var transport http.RoundTripper = http.DefaultTransport
	if cfg.InternalIssuer != "" && cfg.Issuer != "" {
		publicURL, err := url.Parse(cfg.Issuer)
		internalURL, err2 := url.Parse(cfg.InternalIssuer)
		if err == nil && err2 == nil {
			transport = &hostRewritingTransport{
				transport:  http.DefaultTransport,
				targetHost: internalURL.Host,
				publicHost: publicURL.Host,
			}
		}
	}

	httpCli := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	cbSettings := gobreaker.Settings{
		Name:        "oidc-provider",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= maxFail
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Warn().
				Str("circuit_breaker", name).
				Str("from", from.String()).
				Str("to", to.String()).
				Msg("OIDC provider circuit breaker state change")
		},
	}

	return &OIDCHTTPClient{
		cfg:     cfg,
		httpCli: httpCli,
		cb:      gobreaker.NewCircuitBreaker[[]byte](cbSettings),
	}
}

func (c *OIDCHTTPClient) discover(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.discovered {
		return nil
	}

	// 1. If overrides are already fully populated via config, populate cache and skip network call
	if c.cfg.TokenEndpoint != "" && c.cfg.UserInfoEndpoint != "" {
		c.tokenEndpoint = c.cfg.TokenEndpoint
		c.userinfoEndpoint = c.cfg.UserInfoEndpoint
		c.jwksURI = c.cfg.Issuer + "/.well-known/jwks.json"
		c.discovered = true
		return nil
	}

	// 2. If Issuer is empty (e.g. in some minimalist unit tests), fallback to overrides if they are partially set
	if c.cfg.Issuer == "" {
		c.tokenEndpoint = c.cfg.TokenEndpoint
		c.userinfoEndpoint = c.cfg.UserInfoEndpoint
		c.jwksURI = c.cfg.Issuer + "/.well-known/jwks.json"
		c.discovered = true
		return nil
	}

	// 3. Perform OIDC discovery.
	// When InternalIssuer is set we query it instead of the public Issuer so that the BFF,
	// which runs inside the cluster, can reach Keycloak via its cluster-internal Service name
	// rather than the public hostname that is only reachable from the user's browser.
	publicIssuer := strings.TrimSuffix(c.cfg.Issuer, "/")
	discoveryBase := publicIssuer
	if c.cfg.InternalIssuer != "" {
		discoveryBase = strings.TrimSuffix(c.cfg.InternalIssuer, "/")
	}
	discoveryURL := discoveryBase + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return fmt.Errorf("build discovery request: %w", err)
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discovery returned status %d: %s", resp.StatusCode, string(b))
	}

	var config struct {
		TokenEndpoint    string `json:"token_endpoint"`
		UserInfoEndpoint string `json:"userinfo_endpoint"`
		JwksURI          string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return fmt.Errorf("decode discovery response: %w", err)
	}

	// rewriteEndpoint replaces the public issuer prefix with the internal issuer prefix
	// in any discovered endpoint URL. This is necessary because the discovery document
	// returned by Keycloak will reference the public hostname, but the BFF must route
	// all backend-to-Keycloak traffic through the internal cluster service.
	rewriteEndpoint := func(endpoint string) string {
		if c.cfg.InternalIssuer != "" && endpoint != "" && strings.HasPrefix(endpoint, publicIssuer) {
			return strings.TrimSuffix(c.cfg.InternalIssuer, "/") + endpoint[len(publicIssuer):]
		}
		return endpoint
	}

	// Use discovered endpoints, falling back to overrides if they are defined
	if config.TokenEndpoint != "" {
		c.tokenEndpoint = rewriteEndpoint(config.TokenEndpoint)
	} else {
		c.tokenEndpoint = c.cfg.TokenEndpoint
	}

	if config.UserInfoEndpoint != "" {
		c.userinfoEndpoint = rewriteEndpoint(config.UserInfoEndpoint)
	} else {
		c.userinfoEndpoint = c.cfg.UserInfoEndpoint
	}

	if config.JwksURI != "" {
		c.jwksURI = rewriteEndpoint(config.JwksURI)
	} else {
		c.jwksURI = rewriteEndpoint(c.cfg.Issuer + "/.well-known/jwks.json")
	}

	c.discovered = true
	return nil
}

// ExchangeCode exchanges an authorization code for a TokenSet using the OIDC token endpoint.
func (c *OIDCHTTPClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	if err := c.discover(ctx); err != nil {
		return nil, err
	}

	if c.tokenEndpoint == "" {
		return nil, fmt.Errorf("oidc: token endpoint is not configured and discovery failed to retrieve it")
	}

	data, err := c.cb.Execute(func() ([]byte, error) {
		body := url.Values{}
		body.Set("grant_type", "authorization_code")
		body.Set("code", code)
		body.Set("redirect_uri", redirectURI)
		body.Set("client_id", c.cfg.ClientID)
		body.Set("client_secret", c.cfg.ClientSecret)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint, strings.NewReader(body.Encode()))
		if err != nil {
			return nil, fmt.Errorf("oidc: build token request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpCli.Do(req)
		if err != nil {
			return nil, fmt.Errorf("oidc: token endpoint call: %w", err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("oidc: read token response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("oidc: token endpoint returned %d: %s", resp.StatusCode, string(b))
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("oidc: unmarshal token response: %w", err)
	}

	return &outbound.TokenSet{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		IDToken:      raw.IDToken,
		ExpiresIn:    time.Duration(raw.ExpiresIn) * time.Second,
	}, nil
}

// RefreshToken uses the OIDC token endpoint to obtain a fresh token set via the refresh_token grant.
func (c *OIDCHTTPClient) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	if err := c.discover(ctx); err != nil {
		return nil, err
	}

	if c.tokenEndpoint == "" {
		return nil, fmt.Errorf("oidc: token endpoint is not configured and discovery failed to retrieve it")
	}

	data, err := c.cb.Execute(func() ([]byte, error) {
		body := url.Values{}
		body.Set("grant_type", "refresh_token")
		body.Set("refresh_token", refreshToken)
		body.Set("client_id", c.cfg.ClientID)
		body.Set("client_secret", c.cfg.ClientSecret)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint, strings.NewReader(body.Encode()))
		if err != nil {
			return nil, fmt.Errorf("oidc: build refresh request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := c.httpCli.Do(req)
		if err != nil {
			return nil, fmt.Errorf("oidc: refresh token endpoint call: %w", err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("oidc: read refresh response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("oidc: refresh endpoint returned %d: %s", resp.StatusCode, string(b))
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("oidc: unmarshal refresh response: %w", err)
	}

	return &outbound.TokenSet{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		IDToken:      raw.IDToken,
		ExpiresIn:    time.Duration(raw.ExpiresIn) * time.Second,
	}, nil
}

// FetchUserInfo retrieves the OIDC UserInfo payload using the provided access token.
func (c *OIDCHTTPClient) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	if err := c.discover(ctx); err != nil {
		return nil, err
	}

	if c.userinfoEndpoint == "" {
		return nil, fmt.Errorf("oidc: userinfo endpoint is not configured and discovery failed to retrieve it")
	}

	data, err := c.cb.Execute(func() ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userinfoEndpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("oidc: build userinfo request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.httpCli.Do(req)
		if err != nil {
			return nil, fmt.Errorf("oidc: userinfo endpoint call: %w", err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("oidc: read userinfo response: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("oidc: userinfo endpoint returned %d: %s", resp.StatusCode, string(b))
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}

	var payload model.UserInfoPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("oidc: unmarshal userinfo: %w", err)
	}

	return &payload, nil
}

// ValidateLogoutToken parses and cryptographically validates a backchannel logout token.
// It fetches the OIDC provider's JWKS to verify the JWT signature, then extracts the
// sub and sid claims per the OIDC Back-Channel Logout specification.
// The caller should prioritize sid (session-scoped logout) when non-empty.
func (c *OIDCHTTPClient) ValidateLogoutToken(ctx context.Context, rawToken string) (sub string, sessionID string, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	if err := c.discover(ctx); err != nil {
		return "", "", err
	}

	// Step 1: Parse the raw JWT into a JSON Web Signature structure.
	jws, err := jose.ParseSigned(rawToken, []jose.SignatureAlgorithm{jose.RS256, jose.ES256, jose.RS384, jose.ES384})
	if err != nil {
		return "", "", fmt.Errorf("oidc: logout token parse failed: %w", err)
	}

	// Step 2: Verify JWT signature using the OIDC provider's remote JWKS.
	keySet := rp.NewRemoteKeySet(c.httpCli, c.jwksURI)
	payload, err := keySet.VerifySignature(ctx, jws)
	if err != nil {
		return "", "", fmt.Errorf("oidc: logout token signature verification failed: %w", err)
	}

	// Step 2: Unmarshal the verified payload into LogoutTokenClaims.
	var claims zoidc.LogoutTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", "", fmt.Errorf("oidc: logout token claims unmarshal failed: %w", err)
	}

	// Step 3: Validate standard claims (issuer, audience, expiry).
	if claims.Issuer != c.cfg.Issuer {
		return "", "", fmt.Errorf("oidc: logout token issuer mismatch: got %q, want %q", claims.Issuer, c.cfg.Issuer)
	}
	found := false
	for _, aud := range claims.Audience {
		if aud == c.cfg.ClientID {
			found = true
			break
		}
	}
	if !found {
		return "", "", fmt.Errorf("oidc: logout token audience does not contain client_id %q", c.cfg.ClientID)
	}
	if time.Now().After(claims.Expiration.AsTime()) {
		return "", "", fmt.Errorf("oidc: logout token is expired")
	}

	// Step 4: Validate backchannel logout event claim.
	const backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"
	if _, ok := claims.Events[backchannelLogoutEvent]; !ok {
		return "", "", fmt.Errorf("oidc: logout token missing backchannel-logout event claim")
	}

	log.Debug().
		Str("sub", claims.Subject).
		Str("sid", claims.SessionID).
		Msg("Backchannel logout token validated")

	return claims.Subject, claims.SessionID, nil
}

type hostRewritingTransport struct {
	transport  http.RoundTripper
	targetHost string
	publicHost string
}

func (t *hostRewritingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL != nil && req.URL.Host == t.targetHost {
		req = req.Clone(req.Context())
		req.Host = t.publicHost
		req.Header.Set("X-Forwarded-Host", t.publicHost)
		if req.URL.Scheme == "https" {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
	}
	tp := t.transport
	if tp == nil {
		tp = http.DefaultTransport
	}
	return tp.RoundTrip(req)
}

