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

func (m *MockSkillUseCase) CreateCollection(ctx context.Context, collection *model.SkillCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockSkillUseCase) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.SkillCollection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SkillCollection), args.Error(1)
}

func (m *MockSkillUseCase) ListCollections(ctx context.Context) ([]*model.SkillCollection, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.SkillCollection), args.Error(1)
}

func (m *MockSkillUseCase) UpdateCollection(ctx context.Context, collection *model.SkillCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockSkillUseCase) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSkillUseCase) ResolveCollectionSkills(ctx context.Context, id uuid.UUID) ([]*model.Skill, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Skill), args.Error(1)
}

func (m *MockSkillUseCase) AttachCollectionToAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	args := m.Called(ctx, agentID, collectionID)
	return args.Error(0)
}

func (m *MockSkillUseCase) DetachCollectionFromAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	args := m.Called(ctx, agentID, collectionID)
	return args.Error(0)
}

func (m *MockSkillUseCase) ResolveAgentSkills(ctx context.Context, agentID uuid.UUID) ([]*model.ResolvedSkill, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ResolvedSkill), args.Error(1)
}

func (m *MockSkillUseCase) PatchStatus(ctx context.Context, id uuid.UUID, status model.SkillStatus) (*model.Skill, error) {
	args := m.Called(ctx, id, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Skill), args.Error(1)
}

func (m *MockSkillUseCase) AddSkillToCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, collectionID, skillID)
	return args.Error(0)
}

func (m *MockSkillUseCase) RemoveSkillFromCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	args := m.Called(ctx, collectionID, skillID)
	return args.Error(0)
}

func TestSkillHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Skill Successfully Returns Nil", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/skills", handler.Create)

		payload := map[string]interface{}{
			"name":        "web-search",
			"description": "Google search and URL extraction tools",
			"content":     "Dynamic google search guidelines.",
			"status":      "active",
		}

		var capturedCtx context.Context
		var capturedSkillID uuid.UUID
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Skill")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedSkillID = args.Get(1).(*model.Skill).ID
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/skills", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "/api/v1/skills/"+capturedSkillID.String(), resp.Header().Get("Location"))
		assert.Empty(t, resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
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

	t.Run("Update Skill Returns Previous Unmodified State", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/skills/:id", handler.Update)

		id := uuid.New()
		existing := &model.Skill{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "web-search",
			Description: "Original Description",
			Status:      model.SkillStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "web-search-v2",
			"description": "Updated description",
			"content":     "Updated dynamic google search guidelines.",
			"status":      "active",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Skill")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/skills/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "web-search", respBody["name"])
		assert.Equal(t, "Original Description", respBody["description"])
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

		var capturedCtx context.Context
		repo.On("AttachSkillToAgent", mock.Anything, agentID, skillID).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID.String()+"/skills/"+skillID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
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

func TestSkillHandlers_Collections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Collection Successfully Returns Nil", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/skill-collections", handler.CreateCollection)

		sID := uuid.New()
		payload := map[string]interface{}{
			"name":        "general-skills",
			"description": "General assistant capabilities",
			"skills":      []string{sID.String()},
		}

		var capturedCtx context.Context
		var capturedCollectionID uuid.UUID
		repo.On("CreateCollection", mock.Anything, mock.AnythingOfType("*model.SkillCollection")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedCollectionID = args.Get(1).(*model.SkillCollection).ID
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/skill-collections", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "/api/v1/skill-collections/"+capturedCollectionID.String(), resp.Header().Get("Location"))
		assert.Empty(t, resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Get Collection Found", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skill-collections/:id", handler.GetCollectionByID)

		id := uuid.New()
		col := &model.SkillCollection{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "general-skills",
			Description: "General capabilities",
			Skills:      []uuid.UUID{uuid.New()},
		}

		repo.On("GetCollectionByID", mock.Anything, id).Return(col, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skill-collections/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("List Collections Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skill-collections", handler.ListCollections)

		cols := []*model.SkillCollection{
			{
				ID:   uuid.New(),
				Name: "general-skills",
			},
		}

		repo.On("ListCollections", mock.Anything).Return(cols, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skill-collections", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Update Collection Returns Previous Unmodified State", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/skill-collections/:id", handler.UpdateCollection)

		id := uuid.New()
		sID1 := uuid.New()
		sID2 := uuid.New()

		existing := &model.SkillCollection{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "general-skills",
			Description: "Original Description",
			Skills:      []uuid.UUID{sID1},
		}

		payload := map[string]interface{}{
			"name":        "general-skills-v2",
			"description": "Updated Description",
			"skills":      []string{sID1.String(), sID2.String()},
		}

		repo.On("GetCollectionByID", mock.Anything, id).Return(existing, nil)
		repo.On("UpdateCollection", mock.Anything, mock.AnythingOfType("*model.SkillCollection")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/skill-collections/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "general-skills", respBody["name"])
		assert.Equal(t, "Original Description", respBody["description"])
	})

	t.Run("Delete Collection Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/skill-collections/:id", handler.DeleteCollection)

		id := uuid.New()
		repo.On("DeleteCollection", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/skill-collections/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("Resolve Collection Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/skill-collections/:id/resolve", handler.ResolveCollection)

		id := uuid.New()
		skills := []*model.Skill{
			{
				ID:     uuid.New(),
				Name:   "web-search",
				Status: model.SkillStatusActive,
			},
		}

		repo.On("ResolveCollectionSkills", mock.Anything, id).Return(skills, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/skill-collections/"+id.String()+"/resolve", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Patch Skill Status Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PATCH("/api/v1/skills/:id", handler.PatchStatus)

		id := uuid.New()
		updated := &model.Skill{
			ID:     id,
			Name:   "web-search",
			Status: model.SkillStatusSuspended,
		}

		repo.On("PatchStatus", mock.Anything, id, model.SkillStatusSuspended).Return(updated, nil)

		payload := map[string]interface{}{
			"status": "suspended",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/skills/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "suspended", respBody["status"])
	})

	t.Run("Add Skill to Collection Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/skill-collections/:id/skills/:skill_id", handler.AddSkillToCollection)

		collectionID := uuid.New()
		skillID := uuid.New()
		col := &model.SkillCollection{
			ID:   collectionID,
			Name: "general-skills",
		}

		repo.On("AddSkillToCollection", mock.Anything, collectionID, skillID).Return(nil)
		repo.On("GetCollectionByID", mock.Anything, collectionID).Return(col, nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/skill-collections/"+collectionID.String()+"/skills/"+skillID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "general-skills", respBody["name"])
	})

	t.Run("Remove Skill from Collection Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/skill-collections/:id/skills/:skill_id", handler.RemoveSkillFromCollection)

		collectionID := uuid.New()
		skillID := uuid.New()
		col := &model.SkillCollection{
			ID:   collectionID,
			Name: "general-skills",
		}

		repo.On("RemoveSkillFromCollection", mock.Anything, collectionID, skillID).Return(nil)
		repo.On("GetCollectionByID", mock.Anything, collectionID).Return(col, nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/skill-collections/"+collectionID.String()+"/skills/"+skillID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "general-skills", respBody["name"])
	})

	t.Run("Attach Collection to Agent Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/skill-collections/:collection_id", handler.AttachCollectionToAgent)

		agentID := uuid.New()
		collectionID := uuid.New()

		repo.On("AttachCollectionToAgent", mock.Anything, agentID, collectionID).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/agents/"+agentID.String()+"/skill-collections/"+collectionID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "attached")
	})

	t.Run("Detach Collection from Agent Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/agents/:agent_id/skill-collections/:collection_id", handler.DetachCollectionFromAgent)

		agentID := uuid.New()
		collectionID := uuid.New()

		repo.On("DetachCollectionFromAgent", mock.Anything, agentID, collectionID).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/agents/"+agentID.String()+"/skill-collections/"+collectionID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("Get Resolved Agent Skills Successfully", func(t *testing.T) {
		repo := new(MockSkillUseCase)
		handler := NewSkillHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents/:agent_id/skills", handler.GetResolvedSkills)

		agentID := uuid.New()
		resolved := []*model.ResolvedSkill{
			{
				ID:           uuid.New(),
				Name:         "web-search",
				Source:       "collection",
				CollectionID: func() *uuid.UUID { u := uuid.New(); return &u }(),
			},
		}

		repo.On("ResolveAgentSkills", mock.Anything, agentID).Return(resolved, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/agents/"+agentID.String()+"/skills", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), respBody["total"])
		skillsVal := respBody["resolved_skills"].([]interface{})
		assert.Len(t, skillsVal, 1)
		assert.Equal(t, "web-search", skillsVal[0].(map[string]interface{})["name"])
	})
}
