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

	"github.com/google/uuid"
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
func (m *mockSessionUseCase) InitiateLogin(ctx context.Context, redirectTo string) (string, string, error) { return "", "", nil }
func (m *mockSessionUseCase) HandleCallback(ctx context.Context, code, state string) (*model.Session, string, error) { return nil, "", nil }
func (m *mockSessionUseCase) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) { return nil, nil }
func (m *mockSessionUseCase) Logout(ctx context.Context, sessionID string) error { return nil }
func (m *mockSessionUseCase) BackchannelLogout(ctx context.Context, rawLogoutToken string) error { return nil }
func (m *mockSessionUseCase) GetSession(ctx context.Context, sessionID string) (*model.Session, error) { return nil, nil }
func (m *mockSessionUseCase) ValidateAccessToken(ctx context.Context, token string) (*model.UserInfoPayload, error) { return nil, nil }
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
func (m *mockSessionStore) SavePendingState(ctx context.Context, state, redirectTo string, ttl time.Duration) error { return nil }
func (m *mockSessionStore) GetAndDeletePendingState(ctx context.Context, state string) (string, error) { return "", nil }
func (m *mockSessionStore) CacheHTML(ctx context.Context, key string, html string, ttl time.Duration) error { return nil }
func (m *mockSessionStore) GetCachedHTML(ctx context.Context, key string) (string, error) { return "", nil }
var _ outbound.SessionStore = (*mockSessionStore)(nil)

type mockOIDCProvider struct{}
func (m *mockOIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*outbound.TokenSet, error) { return nil, nil }
func (m *mockOIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*outbound.TokenSet, error) { return nil, nil }
func (m *mockOIDCProvider) FetchUserInfo(ctx context.Context, accessToken string) (*model.UserInfoPayload, error) { return nil, nil }
func (m *mockOIDCProvider) ValidateLogoutToken(ctx context.Context, rawToken string) (string, string, error) { return "", "", nil }
func (m *mockOIDCProvider) ValidateAccessToken(ctx context.Context, token string) (*model.UserInfoPayload, error) { return nil, nil }
var _ outbound.OIDCProvider = (*mockOIDCProvider)(nil)

type mockLLMClient struct{}
func (m *mockLLMClient) Create(ctx context.Context, req *outbound.CreateLLMBindingRequest) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Get(ctx context.Context, id uuid.UUID) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateLLMBindingRequest) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockLLMClient) List(ctx context.Context) ([]*outbound.LLMBinding, error) { return nil, nil }

type mockMCPClient struct{}
func (m *mockMCPClient) Create(ctx context.Context, req *outbound.CreateMCPServerRequest) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Get(ctx context.Context, id uuid.UUID) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateMCPServerRequest) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockMCPClient) List(ctx context.Context) ([]*outbound.MCPServer, error) { return nil, nil }

type mockSkillClient struct{}
func (m *mockSkillClient) Create(ctx context.Context, req *outbound.CreateSkillRequest) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateSkillRequest) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockSkillClient) List(ctx context.Context) ([]*outbound.Skill, error) { return nil, nil }

type mockPromptClient struct{}
func (m *mockPromptClient) Create(ctx context.Context, req *outbound.CreatePromptTemplateRequest) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Get(ctx context.Context, id uuid.UUID) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdatePromptTemplateRequest) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockPromptClient) List(ctx context.Context) ([]*outbound.PromptTemplate, error) { return nil, nil }

type mockAgentClient struct{}
func (m *mockAgentClient) Create(ctx context.Context, req *outbound.CreateAgentRequest) (*outbound.Agent, error) { return nil, nil }
func (m *mockAgentClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Agent, error) { return nil, nil }
func (m *mockAgentClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateAgentRequest) (*outbound.Agent, error) { return nil, nil }
func (m *mockAgentClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockAgentClient) List(ctx context.Context) ([]*outbound.Agent, error) { return nil, nil }

type mockCommunityClient struct{}
func (m *mockCommunityClient) Create(ctx context.Context, req *outbound.CreateCommunityRequest) (*outbound.Community, error) { return nil, nil }
func (m *mockCommunityClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Community, error) { return nil, nil }
func (m *mockCommunityClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateCommunityRequest) (*outbound.Community, error) { return nil, nil }
func (m *mockCommunityClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCommunityClient) List(ctx context.Context) ([]*outbound.Community, error) { return nil, nil }
func (m *mockCommunityClient) AssignAgent(ctx context.Context, communityID, agentID uuid.UUID) error { return nil }
func (m *mockCommunityClient) UnassignAgent(ctx context.Context, communityID, agentID uuid.UUID) error { return nil }

type mockKeeperClient struct {
	llm    outbound.LLMBindingClient
	mcp    outbound.MCPServerClient
	skill  outbound.SkillClient
	prompt outbound.PromptClient
	agent  outbound.AgentClient
	comm   outbound.CommunityClient
}
func (m *mockKeeperClient) Ping(ctx context.Context) error { return nil }
func (m *mockKeeperClient) LLMBindings() outbound.LLMBindingClient { return m.llm }
func (m *mockKeeperClient) MCPServers() outbound.MCPServerClient   { return m.mcp }
func (m *mockKeeperClient) Skills() outbound.SkillClient           { return m.skill }
func (m *mockKeeperClient) Prompts() outbound.PromptClient         { return m.prompt }
func (m *mockKeeperClient) Agents() outbound.AgentClient           { return m.agent }
func (m *mockKeeperClient) Communities() outbound.CommunityClient { return m.comm }
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

		// LLM Bindings
		"CreateLLMBinding": {Path: "/llm-bindings", Method: "POST"},
		"GetLLMBinding":    {Path: "/llm-bindings/{id}", Method: "GET"},
		"UpdateLLMBinding": {Path: "/llm-bindings/{id}", Method: "PUT"},
		"DeleteLLMBinding": {Path: "/llm-bindings/{id}", Method: "DELETE"},
		"ListLLMBindings":  {Path: "/llm-bindings", Method: "GET"},

		// MCP Servers (referred to as /mcp-servers in spec, but /mcp-clients in router)
		"CreateMCPServer": {Path: "/mcp-servers", Method: "POST"},
		"GetMCPServer":    {Path: "/mcp-servers/{id}", Method: "GET"},
		"UpdateMCPServer": {Path: "/mcp-servers/{id}", Method: "PUT"},
		"DeleteMCPServer": {Path: "/mcp-servers/{id}", Method: "DELETE"},
		"ListMCPServers":  {Path: "/mcp-servers", Method: "GET"},

		// Skills
		"CreateSkill": {Path: "/skills", Method: "POST"},
		"GetSkill":    {Path: "/skills/{id}", Method: "GET"},
		"UpdateSkill": {Path: "/skills/{id}", Method: "PUT"},
		"DeleteSkill": {Path: "/skills/{id}", Method: "DELETE"},
		"ListSkills":  {Path: "/skills", Method: "GET"},

		// Prompts
		"CreatePrompt": {Path: "/prompts", Method: "POST"},
		"GetPrompt":    {Path: "/prompts/{id}", Method: "GET"},
		"UpdatePrompt": {Path: "/prompts/{id}", Method: "PUT"},
		"DeletePrompt": {Path: "/prompts/{id}", Method: "DELETE"},
		"ListPrompts":  {Path: "/prompts", Method: "GET"},

		// Agents
		"CreateAgent": {Path: "/agents", Method: "POST"},
		"GetAgent":    {Path: "/agents/{id}", Method: "GET"},
		"UpdateAgent": {Path: "/agents/{id}", Method: "PUT"},
		"DeleteAgent": {Path: "/agents/{id}", Method: "DELETE"},
		"ListAgents":  {Path: "/agents", Method: "GET"},

		// Communities
		"CreateCommunity": {Path: "/communities", Method: "POST"},
		"GetCommunity":    {Path: "/communities/{id}", Method: "GET"},
		"UpdateCommunity": {Path: "/communities/{id}", Method: "PUT"},
		"DeleteCommunity": {Path: "/communities/{id}", Method: "DELETE"},
		"ListCommunities":  {Path: "/communities", Method: "GET"},

		// Assignments
		"AssignAgent":   {Path: "/communities/{community_id}/agents/{agent_id}", Method: "POST"},
		"UnassignAgent": {Path: "/communities/{community_id}/agents/{agent_id}", Method: "DELETE"},
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
			normalizedPath := normalizeRoutePath(route.Path)
			key := fmt.Sprintf("%s %s", route.Method, normalizedPath)
			keeperRoutes[key] = true
		}

		for callName, call := range outboundCalls {
			fullPath := call.Path
			fullPath = strings.ReplaceAll(fullPath, "/mcp-servers", "/mcp-clients")
			if fullPath != "/healthz" && !strings.HasPrefix(fullPath, "/api/v1") {
				fullPath = "/api/v1" + fullPath
			}
			normalizedPath := normalizeRoutePath(fullPath)
			routeKey := fmt.Sprintf("%s %s", call.Method, normalizedPath)
			assert.True(t, keeperRoutes[routeKey], "Outbound adapter call %q (%s) does not exist in Keeper router", callName, routeKey)
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
 
	// Extract registered BFF Gin routes under /api/v1 (optionally prefixed by UIPath)
	prefix := cfg.UIPath + "/api/v1"
	ginRoutes := make(map[string]bool)
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, prefix) {
			cleanPath := strings.TrimPrefix(route.Path, cfg.UIPath)
			key := fmt.Sprintf("%s %s", route.Method, normalizeRoutePath(cleanPath))
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
