package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAgentRepository is a mock implementation of inbound.AgentUseCase.
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) Create(ctx context.Context, agent *model.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}

func (m *MockAgentRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}

func (m *MockAgentRepository) List(ctx context.Context) ([]*model.Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Agent), args.Error(1)
}

func (m *MockAgentRepository) Update(ctx context.Context, agent *model.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	args := m.Called(ctx, agentID, communityID)
	return args.Error(0)
}

func (m *MockAgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	args := m.Called(ctx, agentID, communityID)
	return args.Error(0)
}

func TestAgentHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Agent Successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		skillID := uuid.New()
		promptID := uuid.New()
		serverID := uuid.New()

		payload := map[string]interface{}{
			"name":        "qa-agent",
			"description": "Agent for QA tests",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":          2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   3600,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
			"skills":          []string{skillID.String()},
			"prompt_template": promptID.String(),
			"mcp_clients": []map[string]interface{}{
				{
					"client_id": serverID.String(),
				},
			},
		}

		var capturedCtx context.Context
		var capturedAgent *model.Agent
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "/api/v1/agents/"+capturedAgent.ID.String(), resp.Header().Get("Location"))
		assert.Empty(t, resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
		assert.Equal(t, "spoke", capturedAgent.Role)
	})

	t.Run("Create Agent with Hub Role Successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		payload := map[string]interface{}{
			"name":        "hub-agent",
			"description": "Hub agent",
			"role":        "hub",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":         2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   3600,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		var capturedAgent *model.Agent
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "hub", capturedAgent.Role)
		}
	})

	t.Run("Create Agent with Invalid Role Fails", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		payload := map[string]interface{}{
			"name":        "invalid-agent",
			"description": "invalid agent role test",
			"role":        "router",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":         2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   3600,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid agent role")
	})

	t.Run("Create Agent Validation Failure (Missing name)", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		payload := map[string]interface{}{
			"description": "Missing name agent",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})

	t.Run("Create Agent Validation Failure (Invalid brain config)", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		payload := map[string]interface{}{
			"name":        "qa-agent",
			"description": "Agent for QA tests",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":          2048,
				"endpoint":           "not-a-valid-url", // Invalid URL
				"credentials_secret": "",                // Blank
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   3600,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestAgentHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Agent Found", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents/:id", handler.GetByID)

		id := uuid.New()
		agent := &model.Agent{
			ID:          id,
			Name:        "qa-agent",
			Status:      model.AgentStatusDefined,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		repo.On("GetByID", mock.Anything, id).Return(agent, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/agents/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Get Agent Not Found", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents/:id", handler.GetByID)

		id := uuid.New()
		repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("agent not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/agents/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestAgentHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Agents Successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents", handler.List)

		agents := []*model.Agent{
			{
				ID:     uuid.New(),
				Name:   "qa-agent",
				Status: model.AgentStatusDefined,
			},
		}

		repo.On("List", mock.Anything).Return(agents, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("List Agents Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents", handler.List)

		repo.On("List", mock.Anything).Return(([]*model.Agent)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestAgentHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Agent Successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/agents/:id", handler.Update)

		id := uuid.New()
		existing := &model.Agent{
			ID:     id,
			Name:   "qa-agent",
			Status: model.AgentStatusDefined,
			Brain: model.BrainConfig{
				Model:       "gpt-4",
				Temperature: 0.5,
				MaxTokens:   1000,
			},
			ShortTermMemory: model.ShortTermMemoryConfig{
				TTLSeconds: 3600,
			},
			LongTermMemory: model.LongTermMemoryConfig{
				VectorDimension: 1536,
			},
		}

		payload := map[string]interface{}{
			"name":        "qa-agent-updated",
			"description": "Updated Agent description",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":          2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   1800,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		var capturedAgent *model.Agent
		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/agents/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "qa-agent", respBody["name"])
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "spoke", capturedAgent.Role)
		}
	})

	t.Run("Update Agent Role successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/agents/:id", handler.Update)

		id := uuid.New()
		existing := &model.Agent{
			ID:     id,
			Name:   "qa-agent",
			Status: model.AgentStatusDefined,
			Role:   "spoke",
			Brain: model.BrainConfig{
				Model:       "gpt-4",
				Temperature: 0.5,
				MaxTokens:   1000,
			},
			ShortTermMemory: model.ShortTermMemoryConfig{
				TTLSeconds: 3600,
			},
			LongTermMemory: model.LongTermMemoryConfig{
				VectorDimension: 1536,
			},
		}

		payload := map[string]interface{}{
			"name":        "qa-agent-updated",
			"description": "Updated Agent description",
			"role":        "hub",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":          2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   1800,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		var capturedAgent *model.Agent
		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/agents/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "spoke", respBody["role"]) // previousValue returned
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "hub", capturedAgent.Role)
		}
	})

	t.Run("Update Agent with Invalid Role Fails", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/agents/:id", handler.Update)

		id := uuid.New()
		existing := &model.Agent{
			ID:     id,
			Name:   "qa-agent",
			Status: model.AgentStatusDefined,
			Role:   "spoke",
			Brain: model.BrainConfig{
				Model:       "gpt-4",
				Temperature: 0.5,
				MaxTokens:   1000,
			},
			ShortTermMemory: model.ShortTermMemoryConfig{
				TTLSeconds: 3600,
			},
			LongTermMemory: model.LongTermMemoryConfig{
				VectorDimension: 1536,
			},
		}

		payload := map[string]interface{}{
			"name":        "qa-agent-updated",
			"description": "Updated Agent description",
			"role":        "router",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":          2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "qa:short",
				"ttl_seconds":   1800,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "qa-long",
				"vector_dimension": 1536,
			},
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/agents/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid agent role")
	})
}

func TestAgentHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete Agent Successfully", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/agents/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/agents/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}

func TestAgentHandlers_Tier(t *testing.T) {
	gin.SetMode(gin.TestMode)

	basePayload := func() map[string]interface{} {
		return map[string]interface{}{
			"name":        "tier-agent",
			"description": "Agent with tier",
			"brain": map[string]interface{}{
				"model":              "gpt-4o",
				"temperature":        0.7,
				"max_tokens":         2048,
				"endpoint":           "https://api.openai.com/v1",
				"credentials_secret": "secret",
			},
			"short_term_memory": map[string]interface{}{
				"key_namespace": "tier:short",
				"ttl_seconds":   3600,
			},
			"long_term_memory": map[string]interface{}{
				"collection_name":  "tier-long",
				"vector_dimension": 1536,
			},
		}
	}

	t.Run("Create Agent with deployment.tier is saved", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		payload := basePayload()
		payload["deployment"] = map[string]interface{}{
			"tier": "heavy",
		}

		var capturedAgent *model.Agent
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "heavy", capturedAgent.Tier)
		}
	})

	t.Run("Create Agent without deployment block saves empty tier", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents", handler.Create)

		var capturedAgent *model.Agent
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(basePayload())
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "", capturedAgent.Tier)
		}
	})

	t.Run("Update Agent with deployment.tier updates the tier", func(t *testing.T) {
		repo := new(MockAgentRepository)
		handler := NewAgentHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/agents/:id", handler.Update)

		id := uuid.New()
		existing := &model.Agent{
			ID:     id,
			Name:   "tier-agent",
			Status: model.AgentStatusDefined,
			Tier:   "standard",
			Brain: model.BrainConfig{
				Model:       "gpt-4",
				Temperature: 0.5,
				MaxTokens:   1000,
			},
			ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 3600},
			LongTermMemory:  model.LongTermMemoryConfig{VectorDimension: 1536},
		}

		payload := basePayload()
		payload["name"] = "tier-agent"
		payload["deployment"] = map[string]interface{}{
			"tier": "gpu-optimized",
		}

		var capturedAgent *model.Agent
		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Agent")).Return(nil).Run(func(args mock.Arguments) {
			capturedAgent = args.Get(1).(*model.Agent)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/agents/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		if assert.NotNil(t, capturedAgent) {
			assert.Equal(t, "gpu-optimized", capturedAgent.Tier)
		}
	})
}

