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

// MockPromptRepository is a mock implementation of outbound.PromptRepository.
type MockPromptRepository struct {
	mock.Mock
}

func (m *MockPromptRepository) CreateTemplate(ctx context.Context, t *domain.PromptTemplate) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockPromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*domain.PromptTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PromptTemplate), args.Error(1)
}

func (m *MockPromptRepository) GetLatestTemplateByName(ctx context.Context, name string) (*domain.PromptTemplate, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PromptTemplate), args.Error(1)
}

func (m *MockPromptRepository) ListTemplates(ctx context.Context) ([]*domain.PromptTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PromptTemplate), args.Error(1)
}

func (m *MockPromptRepository) ListTemplateVersions(ctx context.Context, name string) ([]*domain.PromptTemplate, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PromptTemplate), args.Error(1)
}

func (m *MockPromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPromptRepository) CreateCollection(ctx context.Context, collection *domain.PromptCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockPromptRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*domain.PromptCollection, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PromptCollection), args.Error(1)
}

func (m *MockPromptRepository) ListCollections(ctx context.Context) ([]*domain.PromptCollection, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PromptCollection), args.Error(1)
}

func (m *MockPromptRepository) UpdateCollection(ctx context.Context, collection *domain.PromptCollection) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockPromptRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*domain.PromptTemplate, error) {
	args := m.Called(ctx, collectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PromptTemplate), args.Error(1)
}

func TestPromptHandlers_CreateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Template Successfully", func(t *testing.T) {
		repo := new(MockPromptRepository)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.POST("/api/v1/prompts", handler.CreateTemplate)

		payload := map[string]interface{}{
			"name":    "system-behavior",
			"content": "You are a chatbot.",
			"role":    "system",
			"status":  "active",
		}

		repo.On("CreateTemplate", mock.Anything, mock.AnythingOfType("*domain.PromptTemplate")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/prompts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "system-behavior", respBody["name"])
		assert.Equal(t, float64(1), respBody["version"])
	})
}

func TestPromptHandlers_UpdateTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Creates New Immutable Version", func(t *testing.T) {
		repo := new(MockPromptRepository)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.PUT("/api/v1/prompts/:id", handler.UpdateTemplate)

		id := uuid.New()
		existing := &domain.PromptTemplate{
			ID:        id,
			Name:      "system-behavior",
			Content:   "Version 1 content",
			Role:      domain.PromptRoleSystem,
			Version:   1,
			Status:    domain.PromptStatusActive,
			CreatedAt: time.Now(),
		}

		payload := map[string]interface{}{
			"content": "Version 2 content",
			"role":    "system",
			"status":  "active",
		}

		repo.On("GetTemplateByID", mock.Anything, id).Return(existing, nil)
		repo.On("GetLatestTemplateByName", mock.Anything, "system-behavior").Return(existing, nil)
		repo.On("CreateTemplate", mock.Anything, mock.AnythingOfType("*domain.PromptTemplate")).Return(nil)

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
		assert.Equal(t, float64(2), respBody["version"]) // Incremented version!
		assert.NotEqual(t, id.String(), respBody["id"])  // New UUID!
	})
}

func TestPromptHandlers_Collections(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Collection Successfully", func(t *testing.T) {
		repo := new(MockPromptRepository)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.POST("/api/v1/prompt-collections", handler.CreateCollection)

		tID := uuid.New()
		payload := map[string]interface{}{
			"name":        "agent-a-prompts",
			"description": "Greeting and system rules",
			"templates":   []string{tID.String()},
		}

		repo.On("CreateCollection", mock.Anything, mock.AnythingOfType("*domain.PromptCollection")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/prompt-collections", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
	})

	t.Run("Resolve Collection Successfully", func(t *testing.T) {
		repo := new(MockPromptRepository)
		handler := NewPromptHandler(repo)

		r := gin.New()
		r.GET("/api/v1/prompt-collections/:id/resolve", handler.ResolveCollection)

		id := uuid.New()
		resolved := []*domain.PromptTemplate{
			{
				ID:      uuid.New(),
				Name:    "system-behavior",
				Content: "You are active.",
				Role:    domain.PromptRoleSystem,
				Version: 2,
				Status:  domain.PromptStatusActive,
			},
		}

		repo.On("ResolveCollectionPrompts", mock.Anything, id).Return(resolved, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/prompt-collections/"+id.String()+"/resolve", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "system-behavior")
	})
}
