//go:build integration

package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/tacito-square/internal/bff"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/morphy76/tacito-square/internal/keeper"
	"github.com/rs/zerolog"
)

// Define mock structures locally so the test compiles independently in test/contract/
type mockSessionUseCase struct{}
func (m *mockSessionUseCase) InitiateLogin(ctx context.Context) (string, string, error) { return "", "", nil }
func (m *mockSessionUseCase) HandleCallback(ctx context.Context, code, state string) (*model.Session, error) { return nil, nil }
func (m *mockSessionUseCase) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) { return nil, nil }
func (m *mockSessionUseCase) Logout(ctx context.Context, sessionID string) error { return nil }
func (m *mockSessionUseCase) BackchannelLogout(ctx context.Context, rawLogoutToken string) error { return nil }
func (m *mockSessionUseCase) GetSession(ctx context.Context, sessionID string) (*model.Session, error) { return nil, nil }
var _ inbound.SessionUseCase = (*mockSessionUseCase)(nil)

type mockEventStreamUseCase struct{}
func (m *mockEventStreamUseCase) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) { return nil, nil }
var _ inbound.EventStreamUseCase = (*mockEventStreamUseCase)(nil)

type mockSessionStore struct{}
func (m *mockSessionStore) Save(ctx context.Context, sess *model.Session, ttl time.Duration) error { return nil }
func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (*model.Session, error) { return nil, nil }
func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error { return nil }
func (m *mockSessionStore) DeleteByUserID(ctx context.Context, userID string) error { return nil }
func (m *mockSessionStore) DeleteByOIDCSessionID(ctx context.Context, issuer, oidcSessionID string) error { return nil }
var _ outbound.SessionStore = (*mockSessionStore)(nil)

type mockOIDCProvider struct{}
func (m *mockOIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) { return nil, nil }
func (m *mockOIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) { return nil, nil }
func (m *mockOIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) { return nil, nil }
func (m *mockOIDCProvider) ValidateLogoutToken(ctx context.Context, rawToken string) (string, string, error) { return "", "", nil }
var _ outbound.OIDCProvider = (*mockOIDCProvider)(nil)

type mockKeeperClient struct{}
func (m *mockKeeperClient) Ping(ctx context.Context) error { return nil }
var _ outbound.KeeperClient = (*mockKeeperClient)(nil)

// TestBFF_KeeperContract_Parity validates that the endpoints requested by the BFF's outbound adapters
// (Keeper HTTP client, SSE client) match the Keeper's OpenAPI specification and the Keeper router.
func TestBFF_KeeperContract_Parity(t *testing.T) {
	// 1. Read Keeper's openapi.json spec
	keeperSpecPath := "../../api/openapi/openapi.json"
	specBytes, err := os.ReadFile(keeperSpecPath)
	require.NoError(t, err, "failed to read Keeper openapi.json at %s", keeperSpecPath)

	var spec OpenAPI
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "failed to parse Keeper openapi.json")

	// 2. Define the paths and methods called by the BFF's outbound adapters
	type outboundCall struct {
		Path   string
		Method string
	}
	outboundCalls := map[string]outboundCall{
		"StreamEvents": {Path: "/events/stream", Method: "GET"},
		"Ping":         {Path: "/healthz", Method: "GET"},
	}

	t.Run("BFF outbound paths exist in Keeper OpenAPI contract", func(t *testing.T) {
		for callName, call := range outboundCalls {
			if call.Path == "/healthz" {
				continue // /healthz is a non-functional endpoint not in openapi.json
			}
			pathObj, exists := spec.Paths[call.Path]
			if assert.True(t, exists, "Outbound adapter call %q targets path %q which is missing from Keeper OpenAPI spec", callName, call.Path) {
				methodLower := strings.ToLower(call.Method)
				_, methodExists := pathObj[methodLower]
				assert.True(t, methodExists, "Outbound adapter call %q uses method %q which is not supported by Keeper path %q in OpenAPI spec", callName, call.Method, call.Path)
			}
		}
	})

	t.Run("BFF outbound calls exist in Keeper router", func(t *testing.T) {
		keeperRouter := keeper.NewServer(nil, nil, nil, nil, zerolog.Nop())
		require.NotNil(t, keeperRouter)

		keeperRoutes := make(map[string]bool)
		for _, route := range keeperRouter.Routes() {
			key := fmt.Sprintf("%s %s", route.Method, route.Path)
			keeperRoutes[key] = true
		}

		for callName, call := range outboundCalls {
			fullPath := call.Path
			if fullPath != "/healthz" && !strings.HasPrefix(fullPath, "/api/v1") {
				fullPath = "/api/v1" + fullPath
			}
			routeKey := fmt.Sprintf("%s %s", call.Method, fullPath)
			assert.True(t, keeperRoutes[routeKey], "Outbound adapter call %q (%s) does not exist in Keeper router: %s", callName, routeKey, routeKey)
		}
	})
}

// TestBFF_RouterOpenAPI_Parity verifies that the BFF Gin router routes match the BFF's bff_openapi.json spec.
func TestBFF_RouterOpenAPI_Parity(t *testing.T) {
	// 1. Read BFF's bff_openapi.json spec
	bffSpecPath := "../../internal/bff/bff_openapi.json"
	specBytes, err := os.ReadFile(bffSpecPath)
	require.NoError(t, err, "failed to read BFF bff_openapi.json at %s", bffSpecPath)

	var spec OpenAPI
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "failed to parse BFF bff_openapi.json")

	// 2. Bootstrap Gin router (dummy config and nil dependencies)
	cfg := bff.Config{Version: "0.1.0", OtelEndpoint: "", LogLevel: "info", GinMode: "test", UIPath: "/ui"}
	router := bff.NewServer(cfg, &mockSessionUseCase{}, &mockEventStreamUseCase{}, &mockSessionStore{}, &mockOIDCProvider{}, &mockKeeperClient{})
	require.NotNil(t, router)
 
	// Extract registered BFF Gin routes under /api/v1
	ginRoutes := make(map[string]bool)
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1") {
			key := fmt.Sprintf("%s %s", route.Method, normalizeRoutePath(route.Path))
			ginRoutes[key] = true
		}
	}
 
	// 3. Extract BFF OpenAPI routes
	openapiRoutes := make(map[string]bool)
	for path, pathObj := range spec.Paths {
		fullPath := path
		if !strings.HasPrefix(fullPath, "/api/v1") {
			fullPath = "/api/v1" + fullPath
		}
		fullPath = normalizeRoutePath(fullPath)
 
		for method := range pathObj {
			upperMethod := strings.ToUpper(method)
			key := fmt.Sprintf("%s %s", upperMethod, fullPath)
			openapiRoutes[key] = true
		}
	}
 
	t.Run("BFF OpenAPI routes exist in BFF Gin router (No Less)", func(t *testing.T) {
		for routeKey := range openapiRoutes {
			assert.True(t, ginRoutes[routeKey], "Route defined in BFF OpenAPI does not exist in BFF Gin router: %s", routeKey)
		}
	})
 
	t.Run("BFF Gin routes exist in BFF OpenAPI (No More)", func(t *testing.T) {
		for routeKey := range ginRoutes {
			assert.True(t, openapiRoutes[routeKey], "Route registered in BFF Gin router does not exist in BFF OpenAPI: %s", routeKey)
		}
	})
}
