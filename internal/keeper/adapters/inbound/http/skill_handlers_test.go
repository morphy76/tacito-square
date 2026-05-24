package http

import (
	"bytes"
	"context"
	"encoding/json"
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

// MockSkillUseCase is a mock implementation of inbound.SkillUseCase.
type MockSkillUseCase struct {
	mock.Mock
}

func (m *MockSkillUseCase) Create(ctx context.Context, skill *model.Skill) error {
	args := m.Called(ctx, skill)
	return args.Error(0)
}

func (m *MockSkillUseCase) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Skill), args.Error(1)
}

func (m *MockSkillUseCase) GetByName(ctx context.Context, name string) (*model.Skill, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Skill), args.Error(1)
}

func (m *MockSkillUseCase) List(ctx context.Context) ([]*model.Skill, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Skill), args.Error(1)
}

func (m *MockSkillUseCase) Update(ctx context.Context, skill *model.Skill) error {
	args := m.Called(ctx, skill)
	return args.Error(0)
}

func (m *MockSkillUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSkillUseCase) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, agentID, skillID)
	return args.Error(0)
}

func (m *MockSkillUseCase) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, agentID, skillID)
	return args.Error(0)
}

func (m *MockSkillUseCase) ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Skill, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Skill), args.Error(1)
}

func TestSkillHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Skill Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/skills", handler.Create)

		mcpID := uuid.New()
		payload := map[string]interface{}{
			"name":          "web-search",
			"description":   "Google search and URL extraction tools",
			"mcp_servers":   []string{mcpID.String()},
			"allowed_tools": []string{"search_google", "fetch_url"},
			"denied_tools":  []string{"format_disk"},
		}

		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Skill")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "web-search", respBody["name"])
		assert.NotEmpty(t, respBody["id"])
	})

	t.Run("Create Skill Validation Failure (Missing name)", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/skills", handler.Create)

		payload := map[string]interface{}{
			"description": "Missing name skill",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestSkillHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Skill Found", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skills/:id", handler.GetByID)

		id := uuid.New()
		skill := &model.Skill{
			ID:          id,
			Name:        "web-search",
			Status:      model.SkillStatusActive,
			MCPServers:  []uuid.UUID{uuid.New()},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		repo.On("GetByID", mock.Anything, id).Return(skill, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skills/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestSkillHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Skills Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skills", handler.List)

		skills := []*model.Skill{
			{
				ID:     uuid.New(),
				Name:   "web-search",
				Status: model.SkillStatusActive,
			},
		}

		repo.On("List", mock.Anything).Return(skills, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skills", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("List Skills Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skills", handler.List)

		repo.On("List", mock.Anything).Return(([]*model.Skill)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skills", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestSkillHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Skill Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/skills/:id", handler.Update)

		id := uuid.New()
		existing := &model.Skill{
			ID:     id,
			Name:   "web-search",
			Status: model.SkillStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "web-search-v2",
			"description": "Updated description",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Skill")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/skills/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestSkillHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete Skill Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/skills/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/skills/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}

func TestSkillHandlers_AgentAssociations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Attach Skill to Agent Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/skills/:skill_id", handler.AttachSkillToAgent)

		agentID := uuid.New()
		skillID := uuid.New()

		repo.On("AttachSkillToAgent", mock.Anything, agentID, skillID).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID.String()+"/skills/"+skillID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Detach Skill from Agent Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/agents/:agent_id/skills/:skill_id", handler.DetachSkillFromAgent)

		agentID := uuid.New()
		skillID := uuid.New()

		repo.On("DetachSkillFromAgent", mock.Anything, agentID, skillID).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID.String()+"/skills/"+skillID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}
