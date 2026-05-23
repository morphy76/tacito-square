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
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSkillRepository is a mock implementation of outbound.SkillRepository.
type MockSkillRepository struct {
	mock.Mock
}

func (m *MockSkillRepository) Create(ctx context.Context, skill *domain.Skill) error {
	args := m.Called(ctx, skill)
	return args.Error(0)
}

func (m *MockSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Skill, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Skill), args.Error(1)
}

func (m *MockSkillRepository) GetByName(ctx context.Context, name string) (*domain.Skill, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Skill), args.Error(1)
}

func (m *MockSkillRepository) List(ctx context.Context) ([]*domain.Skill, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Skill), args.Error(1)
}

func (m *MockSkillRepository) Update(ctx context.Context, skill *domain.Skill) error {
	args := m.Called(ctx, skill)
	return args.Error(0)
}

func (m *MockSkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSkillRepository) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, agentID, skillID)
	return args.Error(0)
}

func (m *MockSkillRepository) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, agentID, skillID)
	return args.Error(0)
}

func (m *MockSkillRepository) ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Skill, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Skill), args.Error(1)
}

func TestSkillHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Skill Successfully", func(t *testing.T) {
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.POST("/api/v1/skills", handler.Create)

		mcpID := uuid.New()
		payload := map[string]interface{}{
			"name":          "web-search",
			"description":   "Google search and URL extraction tools",
			"mcp_servers":   []string{mcpID.String()},
			"allowed_tools": []string{"search_google", "fetch_url"},
			"denied_tools":  []string{"format_disk"},
		}

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Skill")).Return(nil)

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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.GET("/api/v1/skills/:id", handler.GetByID)

		id := uuid.New()
		skill := &domain.Skill{
			ID:          id,
			Name:        "web-search",
			Status:      domain.SkillStatusActive,
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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.GET("/api/v1/skills", handler.List)

		skills := []*domain.Skill{
			{
				ID:     uuid.New(),
				Name:   "web-search",
				Status: domain.SkillStatusActive,
			},
		}

		repo.On("List", mock.Anything).Return(skills, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skills", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestSkillHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Skill Successfully", func(t *testing.T) {
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.PUT("/api/v1/skills/:id", handler.Update)

		id := uuid.New()
		existing := &domain.Skill{
			ID:     id,
			Name:   "web-search",
			Status: domain.SkillStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "web-search-v2",
			"description": "Updated description",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Skill")).Return(nil)

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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
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
		repo := new(MockSkillRepository)
		handler := NewSkillHandler(repo)

		r := gin.New()
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
