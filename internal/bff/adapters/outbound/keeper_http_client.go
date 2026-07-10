package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Compile-time interface satisfaction assertion.
var _ outbound.KeeperClient = (*KeeperHTTPClient)(nil)

// KeeperClientConfig holds configurable parameters for the Keeper outbound adapter.
type KeeperClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// KeeperHTTPClient is a driven adapter implementing outbound.KeeperClient,
// issuing HTTP calls to the Keeper core service with OTel trace propagation.
type KeeperHTTPClient struct {
	cfg         KeeperClientConfig
	client      *http.Client
	llmBindings outbound.LLMBindingClient
	mcpServers  outbound.MCPServerClient
	skills      outbound.SkillClient
	prompts     outbound.PromptClient
	agents      outbound.AgentClient
	communities outbound.CommunityClient
}

// NewKeeperHTTPClient constructs a new KeeperHTTPClient with an OTel-instrumented HTTP transport and sub-clients.
func NewKeeperHTTPClient(cfg KeeperClientConfig) *KeeperHTTPClient {
	k := &KeeperHTTPClient{
		cfg: cfg,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   cfg.Timeout,
		},
	}
	k.llmBindings = &llmBindingSubClient{client: k}
	k.mcpServers = &mcpServerSubClient{client: k}
	k.skills = &skillSubClient{client: k}
	k.prompts = &promptSubClient{client: k}
	k.agents = &agentSubClient{client: k}
	k.communities = &communitySubClient{client: k}
	return k
}

// Ping verifies connectivity to the Keeper service by calling its /healthz endpoint.
func (k *KeeperHTTPClient) Ping(ctx context.Context) error {
	return k.doRequest(ctx, http.MethodGet, "/healthz", nil, nil)
}

func (k *KeeperHTTPClient) LLMBindings() outbound.LLMBindingClient { return k.llmBindings }
func (k *KeeperHTTPClient) MCPServers() outbound.MCPServerClient   { return k.mcpServers }
func (k *KeeperHTTPClient) Skills() outbound.SkillClient           { return k.skills }
func (k *KeeperHTTPClient) Prompts() outbound.PromptClient         { return k.prompts }
func (k *KeeperHTTPClient) Agents() outbound.AgentClient           { return k.agents }
func (k *KeeperHTTPClient) Communities() outbound.CommunityClient { return k.communities }

// doRequest is a DRY helper to perform context-aware, tenant-isolated, OTel-instrumented HTTP requests to Keeper.
func (k *KeeperHTTPClient) doRequest(ctx context.Context, method, path string, requestBody interface{}, responseVal interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, k.cfg.Timeout)
	defer cancel()

	var bodyReader io.Reader
	if requestBody != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(requestBody); err != nil {
			return fmt.Errorf("keeper client: serialize request body: %w", err)
		}
		bodyReader = &buf
	}

	url := k.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("keeper client: build request: %w", err)
	}

	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Propagate tenant identity via HTTP headers
	if t := tenant.FromContext(ctx); t != nil {
		if t.TenantID != "" {
			req.Header.Set("X-Tenant-ID", t.TenantID)
		}
		if t.SubscriptionID != "" {
			req.Header.Set("X-Subscription-ID", t.SubscriptionID)
		}
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("keeper client: execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errPayload struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errPayload); err == nil && errPayload.Error != "" {
			return fmt.Errorf("keeper client: request returned %d: %s", resp.StatusCode, errPayload.Error)
		}
		return fmt.Errorf("keeper client: request returned status %d", resp.StatusCode)
	}

	if responseVal != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseVal); err != nil {
			return fmt.Errorf("keeper client: deserialize response: %w", err)
		}
	}

	return nil
}

// --- LLM Binding Sub-Client ---

type llmBindingSubClient struct {
	client *KeeperHTTPClient
}

func (s *llmBindingSubClient) Create(ctx context.Context, req *outbound.CreateLLMBindingRequest) (*outbound.LLMBinding, error) {
	var resp outbound.LLMBinding
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/llm-bindings", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *llmBindingSubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.LLMBinding, error) {
	var resp outbound.LLMBinding
	path := fmt.Sprintf("/api/v1/llm-bindings/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *llmBindingSubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateLLMBindingRequest) (*outbound.LLMBinding, error) {
	var resp outbound.LLMBinding
	path := fmt.Sprintf("/api/v1/llm-bindings/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *llmBindingSubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/llm-bindings/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *llmBindingSubClient) List(ctx context.Context) ([]*outbound.LLMBinding, error) {
	var resp []*outbound.LLMBinding
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/llm-bindings", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- MCP Server Sub-Client ---

type mcpServerSubClient struct {
	client *KeeperHTTPClient
}

func (s *mcpServerSubClient) Create(ctx context.Context, req *outbound.CreateMCPServerRequest) (*outbound.MCPServer, error) {
	var resp outbound.MCPServer
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/mcp-clients", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *mcpServerSubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.MCPServer, error) {
	var resp outbound.MCPServer
	path := fmt.Sprintf("/api/v1/mcp-clients/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *mcpServerSubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateMCPServerRequest) (*outbound.MCPServer, error) {
	var resp outbound.MCPServer
	path := fmt.Sprintf("/api/v1/mcp-clients/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *mcpServerSubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/mcp-clients/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *mcpServerSubClient) List(ctx context.Context) ([]*outbound.MCPServer, error) {
	var resp []*outbound.MCPServer
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/mcp-clients", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- Skill Sub-Client ---

type skillSubClient struct {
	client *KeeperHTTPClient
}

func (s *skillSubClient) Create(ctx context.Context, req *outbound.CreateSkillRequest) (*outbound.Skill, error) {
	var resp outbound.Skill
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/skills", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *skillSubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Skill, error) {
	var resp outbound.Skill
	path := fmt.Sprintf("/api/v1/skills/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *skillSubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateSkillRequest) (*outbound.Skill, error) {
	var resp outbound.Skill
	path := fmt.Sprintf("/api/v1/skills/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *skillSubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/skills/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *skillSubClient) List(ctx context.Context) ([]*outbound.Skill, error) {
	var resp []*outbound.Skill
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/skills", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- Prompt Sub-Client ---

type promptSubClient struct {
	client *KeeperHTTPClient
}

func (s *promptSubClient) Create(ctx context.Context, req *outbound.CreatePromptTemplateRequest) (*outbound.PromptTemplate, error) {
	var resp outbound.PromptTemplate
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/prompts", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *promptSubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.PromptTemplate, error) {
	var resp outbound.PromptTemplate
	path := fmt.Sprintf("/api/v1/prompts/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *promptSubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdatePromptTemplateRequest) (*outbound.PromptTemplate, error) {
	var resp outbound.PromptTemplate
	path := fmt.Sprintf("/api/v1/prompts/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *promptSubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/prompts/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *promptSubClient) List(ctx context.Context) ([]*outbound.PromptTemplate, error) {
	var resp []*outbound.PromptTemplate
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/prompts", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- Agent Sub-Client ---

type agentSubClient struct {
	client *KeeperHTTPClient
}

func (s *agentSubClient) Create(ctx context.Context, req *outbound.CreateAgentRequest) (*outbound.Agent, error) {
	var resp outbound.Agent
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/agents", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *agentSubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Agent, error) {
	var resp outbound.Agent
	path := fmt.Sprintf("/api/v1/agents/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *agentSubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateAgentRequest) (*outbound.Agent, error) {
	var resp outbound.Agent
	path := fmt.Sprintf("/api/v1/agents/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *agentSubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/agents/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *agentSubClient) List(ctx context.Context) ([]*outbound.Agent, error) {
	var resp []*outbound.Agent
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/agents", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- Community Sub-Client ---

type communitySubClient struct {
	client *KeeperHTTPClient
}

func (s *communitySubClient) Create(ctx context.Context, req *outbound.CreateCommunityRequest) (*outbound.Community, error) {
	var resp outbound.Community
	if err := s.client.doRequest(ctx, http.MethodPost, "/api/v1/communities", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *communitySubClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Community, error) {
	var resp outbound.Community
	path := fmt.Sprintf("/api/v1/communities/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *communitySubClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateCommunityRequest) (*outbound.Community, error) {
	var resp outbound.Community
	path := fmt.Sprintf("/api/v1/communities/%s", id.String())
	if err := s.client.doRequest(ctx, http.MethodPut, path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *communitySubClient) Delete(ctx context.Context, id uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/communities/%s", id.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}

func (s *communitySubClient) List(ctx context.Context) ([]*outbound.Community, error) {
	var resp []*outbound.Community
	if err := s.client.doRequest(ctx, http.MethodGet, "/api/v1/communities", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *communitySubClient) AssignAgent(ctx context.Context, communityID, agentID uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/communities/%s/agents/%s", communityID.String(), agentID.String())
	return s.client.doRequest(ctx, http.MethodPost, path, nil, nil)
}

func (s *communitySubClient) UnassignAgent(ctx context.Context, communityID, agentID uuid.UUID) error {
	path := fmt.Sprintf("/api/v1/communities/%s/agents/%s", communityID.String(), agentID.String())
	return s.client.doRequest(ctx, http.MethodDelete, path, nil, nil)
}
