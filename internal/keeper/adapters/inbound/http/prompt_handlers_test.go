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

// MockPromptUseCase is a mock implementation of inbound.PromptUseCase.
type MockPromptUseCase struct {
	mock.Mock
}

func (m *MockPromptUseCase) CreateTemplate(ctx context.Context, t *model.PromptTemplate) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockPromptUseCase) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PromptTemplate), args.Error(1)
}

func (m *MockPromptUseCase) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptTemplate), args.Error(1)
}

func (m *MockPromptUseCase) UpdateTemplate(ctx context.Context, t *model.PromptTemplate) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockPromptUseCase) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPromptUseCase) CreateCollection(ctx context.Context, collection *model.PromptCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockPromptUseCase) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PromptCollection), args.Error(1)
}

func (m *MockPromptUseCase) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptCollection), args.Error(1)
}

func (m *MockPromptUseCase) UpdateCollection(ctx context.Context, collection *model.PromptCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockPromptUseCase) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPromptUseCase) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	args := m.Called(ctx, collectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.PromptTemplate), args.Error(1)
}



func TestPromptHandlers_CreateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Template Successfully Returns Nil", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/prompts", handler.CreateTemplate)

		payload := map[string]interface{}{
			"name":    "system-behavior",
			"content": "You are a chatbot.",
			"status":  "active",
		}

		repo.On("CreateTemplate", mock.Anything, mock.AnythingOfType("*model.PromptTemplate")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "null", resp.Body.String())
	})
}

func TestPromptHandlers_UpdateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Returns Previous Unmodified State", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/prompts/:id", handler.UpdateTemplate)

		id := uuid.New()
		existing := &model.PromptTemplate{
			ID:        id,
			TenantID:  "test-tenant.com",
			Name:      "system-behavior",
			Content:   "Original Content",
			Status:    model.PromptStatusActive,
			CreatedAt: time.Now(),
		}

		payload := map[string]interface{}{
			"name":    "system-behavior-updated",
			"content": "Updated Content",
			"status":  "active",
		}

		repo.On("GetTemplateByID", mock.Anything, id).Return(existing, nil)
		repo.On("UpdateTemplate", mock.Anything, mock.AnythingOfType("*model.PromptTemplate")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/prompts/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "system-behavior", respBody["name"])
		assert.Equal(t, "Original Content", respBody["content"]) // Returns previous state!
	})
}

func TestPromptHandlers_Collections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Collection Successfully Returns Nil", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/prompt-collections", handler.CreateCollection)

		tID := uuid.New()
		payload := map[string]interface{}{
			"name":        "agent-a-prompts",
			"description": "Greeting and system rules",
			"templates":   []string{tID.String()},
		}

		repo.On("CreateCollection", mock.Anything, mock.AnythingOfType("*model.PromptCollection")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/prompt-collections", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "null", resp.Body.String())
	})

	t.Run("Resolve Collection Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/prompt-collections/:id/resolve", handler.ResolveCollection)

		id := uuid.New()
		repo.On("ResolveCollectionPrompts", mock.Anything, id).Return(([]*model.PromptTemplate)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/prompt-collections/"+id.String()+"/resolve", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestPromptHandlers_ListTemplates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Templates Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/prompts", handler.ListTemplates)

		repo.On("ListTemplates", mock.Anything).Return(([]*model.PromptTemplate)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/prompts", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestPromptHandlers_ListCollections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Collections Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockPromptUseCase)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/prompt-collections", handler.ListCollections)

		repo.On("ListCollections", mock.Anything).Return(([]*model.PromptCollection)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/prompt-collections", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}
