package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bffhttp "github.com/morphy76/tacito-square/internal/bff/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/bff/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

type mockKeeperClient struct {
	pingErr error
	llm     *mockLLMClient
	mcp     *mockMCPClient
	skill   *mockSkillClient
	prompt  *mockPromptClient
	agent   *mockAgentClient
	comm    *mockCommunityClient
}

func (m *mockKeeperClient) Ping(ctx context.Context) error { return m.pingErr }
func (m *mockKeeperClient) LLMBindings() outbound.LLMBindingClient { return m.llm }
func (m *mockKeeperClient) MCPServers() outbound.MCPServerClient   { return m.mcp }
func (m *mockKeeperClient) Skills() outbound.SkillClient           { return m.skill }
func (m *mockKeeperClient) Prompts() outbound.PromptClient         { return m.prompt }
func (m *mockKeeperClient) Agents() outbound.AgentClient           { return m.agent }
func (m *mockKeeperClient) Communities() outbound.CommunityClient { return m.comm }

type mockLLMClient struct {
	ListFunc func(ctx context.Context) ([]*outbound.LLMBinding, error)
}
func (m *mockLLMClient) Create(ctx context.Context, req *outbound.CreateLLMBindingRequest) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Get(ctx context.Context, id uuid.UUID) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateLLMBindingRequest) (*outbound.LLMBinding, error) { return nil, nil }
func (m *mockLLMClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockLLMClient) List(ctx context.Context) ([]*outbound.LLMBinding, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

type mockMCPClient struct {
	ListFunc func(ctx context.Context) ([]*outbound.MCPServer, error)
}
func (m *mockMCPClient) Create(ctx context.Context, req *outbound.CreateMCPServerRequest) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Get(ctx context.Context, id uuid.UUID) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateMCPServerRequest) (*outbound.MCPServer, error) { return nil, nil }
func (m *mockMCPClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockMCPClient) List(ctx context.Context) ([]*outbound.MCPServer, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

type mockSkillClient struct {
	ListFunc func(ctx context.Context) ([]*outbound.Skill, error)
}
func (m *mockSkillClient) Create(ctx context.Context, req *outbound.CreateSkillRequest) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateSkillRequest) (*outbound.Skill, error) { return nil, nil }
func (m *mockSkillClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockSkillClient) List(ctx context.Context) ([]*outbound.Skill, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

type mockPromptClient struct {
	ListFunc func(ctx context.Context) ([]*outbound.PromptTemplate, error)
}
func (m *mockPromptClient) Create(ctx context.Context, req *outbound.CreatePromptTemplateRequest) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Get(ctx context.Context, id uuid.UUID) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdatePromptTemplateRequest) (*outbound.PromptTemplate, error) { return nil, nil }
func (m *mockPromptClient) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockPromptClient) List(ctx context.Context) ([]*outbound.PromptTemplate, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

type mockAgentClient struct {
	ListFunc   func(ctx context.Context) ([]*outbound.Agent, error)
	GetFunc    func(ctx context.Context, id uuid.UUID) (*outbound.Agent, error)
	CreateFunc func(ctx context.Context, req *outbound.CreateAgentRequest) (*outbound.Agent, error)
	UpdateFunc func(ctx context.Context, id uuid.UUID, req *outbound.UpdateAgentRequest) (*outbound.Agent, error)
	DeleteFunc func(ctx context.Context, id uuid.UUID) error
}
func (m *mockAgentClient) Create(ctx context.Context, req *outbound.CreateAgentRequest) (*outbound.Agent, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}
func (m *mockAgentClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Agent, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAgentClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateAgentRequest) (*outbound.Agent, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}
func (m *mockAgentClient) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockAgentClient) List(ctx context.Context) ([]*outbound.Agent, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

type mockCommunityClient struct {
	ListFunc          func(ctx context.Context) ([]*outbound.Community, error)
	GetFunc           func(ctx context.Context, id uuid.UUID) (*outbound.Community, error)
	CreateFunc        func(ctx context.Context, req *outbound.CreateCommunityRequest) (*outbound.Community, error)
	UpdateFunc        func(ctx context.Context, id uuid.UUID, req *outbound.UpdateCommunityRequest) (*outbound.Community, error)
	DeleteFunc        func(ctx context.Context, id uuid.UUID) error
	AssignAgentFunc   func(ctx context.Context, communityID, agentID uuid.UUID) error
	UnassignAgentFunc func(ctx context.Context, communityID, agentID uuid.UUID) error
}
func (m *mockCommunityClient) Create(ctx context.Context, req *outbound.CreateCommunityRequest) (*outbound.Community, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}
func (m *mockCommunityClient) Get(ctx context.Context, id uuid.UUID) (*outbound.Community, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockCommunityClient) Update(ctx context.Context, id uuid.UUID, req *outbound.UpdateCommunityRequest) (*outbound.Community, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}
func (m *mockCommunityClient) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockCommunityClient) List(ctx context.Context) ([]*outbound.Community, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}
func (m *mockCommunityClient) AssignAgent(ctx context.Context, communityID, agentID uuid.UUID) error {
	if m.AssignAgentFunc != nil {
		return m.AssignAgentFunc(ctx, communityID, agentID)
	}
	return nil
}
func (m *mockCommunityClient) UnassignAgent(ctx context.Context, communityID, agentID uuid.UUID) error {
	if m.UnassignAgentFunc != nil {
		return m.UnassignAgentFunc(ctx, communityID, agentID)
	}
	return nil
}

func TestRequireRoles_AccessDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return &model.Session{
				ID:       "sess-1",
				UserID:   "user-1",
				TenantID: "tenant.com",
				UserInfo: model.UserInfoPayload{
					Roles: []string{"viewer"},
				},
			}, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.Use(bffhttp.RequireRoles("keeper-admin", "agent-spawner"))
	r.GET("/configurator/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/configurator/test", nil)
	req.AddCookie(&http.Cookie{Name: "bff_session_id", Value: "sess-1"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "Insufficient permissions", body["error"])
}

func TestRequireRoles_AccessAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return &model.Session{
				ID:       "sess-1",
				UserID:   "user-1",
				TenantID: "tenant.com",
				UserInfo: model.UserInfoPayload{
					Roles: []string{"agent-spawner"},
				},
			}, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))
	r.Use(bffhttp.RequireRoles("keeper-admin", "agent-spawner"))
	r.GET("/configurator/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/configurator/test", nil)
	req.AddCookie(&http.Cookie{Name: "bff_session_id", Value: "sess-1"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantContextPropagation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockUC := &mockSessionUseCase{
		GetSessionFunc: func(ctx context.Context, sessionID string) (*model.Session, error) {
			return &model.Session{
				ID:       "sess-1",
				UserID:   "user-1",
				TenantID: "tenant.com",
				UserInfo: model.UserInfoPayload{
					TenantID:       "tenant.com",
					SubscriptionID: "sub-123",
					Roles:          []string{"keeper-admin"},
				},
			}, nil
		},
	}

	r.Use(bffhttp.SessionMiddleware(mockUC, "/ui"))

	var propagatedTenant *tenant.Tenant
	r.GET("/test", func(c *gin.Context) {
		propagatedTenant = tenant.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "bff_session_id", Value: "sess-1"})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, propagatedTenant)
	assert.Equal(t, "tenant.com", propagatedTenant.TenantID)
	assert.Equal(t, "sub-123", propagatedTenant.SubscriptionID)
}

func TestGetWizardOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockKeeper := &mockKeeperClient{
		llm: &mockLLMClient{
			ListFunc: func(ctx context.Context) ([]*outbound.LLMBinding, error) {
				return []*outbound.LLMBinding{{Name: "binding-1"}}, nil
			},
		},
		skill: &mockSkillClient{
			ListFunc: func(ctx context.Context) ([]*outbound.Skill, error) {
				return []*outbound.Skill{{Name: "skill-1"}}, nil
			},
		},
		prompt: &mockPromptClient{
			ListFunc: func(ctx context.Context) ([]*outbound.PromptTemplate, error) {
				return []*outbound.PromptTemplate{{Name: "prompt-1"}}, nil
			},
		},
	}

	handler := bffhttp.NewConfiguratorHandler(mockKeeper)
	r.GET("/wizard/options", handler.GetWizardOptions)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/wizard/options", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Contains(t, resp, "llm_bindings")
	assert.Contains(t, resp, "skills")
	assert.Contains(t, resp, "prompts")
}

func TestConfiguratorHandler_CRUDAndSync(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	agentID := uuid.New()
	commID := uuid.New()

	mockKeeper := &mockKeeperClient{
		agent: &mockAgentClient{
			ListFunc: func(ctx context.Context) ([]*outbound.Agent, error) {
				return []*outbound.Agent{
					{ID: agentID, Name: "agent-1", CommunityID: &commID},
				}, nil
			},
			GetFunc: func(ctx context.Context, id uuid.UUID) (*outbound.Agent, error) {
				return &outbound.Agent{ID: id, Name: "agent-1"}, nil
			},
			CreateFunc: func(ctx context.Context, req *outbound.CreateAgentRequest) (*outbound.Agent, error) {
				return &outbound.Agent{ID: agentID, Name: req.Name}, nil
			},
			UpdateFunc: func(ctx context.Context, id uuid.UUID, req *outbound.UpdateAgentRequest) (*outbound.Agent, error) {
				return &outbound.Agent{ID: id, Name: req.Name}, nil
			},
			DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		},
		comm: &mockCommunityClient{
			ListFunc: func(ctx context.Context) ([]*outbound.Community, error) {
				return []*outbound.Community{
					{ID: commID, Name: "comm-1"},
				}, nil
			},
			GetFunc: func(ctx context.Context, id uuid.UUID) (*outbound.Community, error) {
				return &outbound.Community{ID: id, Name: "comm-1"}, nil
			},
			CreateFunc: func(ctx context.Context, req *outbound.CreateCommunityRequest) (*outbound.Community, error) {
				return &outbound.Community{ID: commID, Name: req.Name}, nil
			},
			UpdateFunc: func(ctx context.Context, id uuid.UUID, req *outbound.UpdateCommunityRequest) (*outbound.Community, error) {
				return &outbound.Community{ID: id, Name: req.Name}, nil
			},
			DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
			AssignAgentFunc: func(ctx context.Context, communityID, agentID uuid.UUID) error {
				return nil
			},
			UnassignAgentFunc: func(ctx context.Context, communityID, agentID uuid.UUID) error {
				return nil
			},
		},
	}

	handler := bffhttp.NewConfiguratorHandler(mockKeeper)

	// Register test routes
	r.GET("/agents", handler.ListAgents)
	r.GET("/agents/:id", handler.GetAgent)
	r.POST("/agents", handler.CreateAgent)
	r.PUT("/agents/:id", handler.UpdateAgent)
	r.DELETE("/agents/:id", handler.DeleteAgent)

	r.GET("/communities", handler.ListCommunities)
	r.GET("/communities/:id", handler.GetCommunity)
	r.POST("/communities", handler.CreateCommunity)
	r.PUT("/communities/:id", handler.UpdateCommunity)
	r.DELETE("/communities/:id", handler.DeleteCommunity)

	r.POST("/communities/:id/agents/:agent_id", handler.AssignAgent)
	r.DELETE("/communities/:id/agents/:agent_id", handler.UnassignAgent)

	r.POST("/advanced-sync", handler.AdvancedSync)

	t.Run("Agents CRUD", func(t *testing.T) {
		// List
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/agents", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var list []*outbound.Agent
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		assert.Len(t, list, 1)

		// Get
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodGet, "/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Create
		cReq := outbound.CreateAgentRequest{Name: "new-agent"}
		body, _ := json.Marshal(cReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/agents", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Update
		uReq := outbound.UpdateAgentRequest{Name: "updated-agent"}
		body, _ = json.Marshal(uReq)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPut, "/agents/"+agentID.String(), bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Delete
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodDelete, "/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Communities CRUD & Assignments", func(t *testing.T) {
		// List (verify enrichment)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/communities", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var list []map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		assert.Len(t, list, 1)
		assert.Contains(t, list[0], "agents")
		agentsList := list[0]["agents"].([]interface{})
		assert.Len(t, agentsList, 1)
		assert.Equal(t, agentID.String(), agentsList[0])

		// Assignment Independent endpoints
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/communities/%s/agents/%s", commID, agentID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)

		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/communities/%s/agents/%s", commID, agentID), nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("Advanced Sync Agents", func(t *testing.T) {
		syncAgents := []map[string]interface{}{
			{
				"id":   agentID.String(),
				"name": "agent-synced",
				"brain": map[string]interface{}{
					"llm_binding_id": uuid.New().String(),
				},
				"short_term_memory": map[string]interface{}{
					"key_namespace": "test",
					"ttl_seconds":   100,
				},
				"long_term_memory": map[string]interface{}{
					"collection_name":  "test",
					"vector_dimension": 128,
				},
			},
		}
		body, _ := json.Marshal(syncAgents)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/advanced-sync", bytes.NewReader(body))
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
