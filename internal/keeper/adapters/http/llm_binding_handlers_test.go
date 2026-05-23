package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func testTenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ten, _ := tenant.New("test-tenant.com", "")
		ctx := tenant.ContextWithTenant(c.Request.Context(), ten)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// MockLLMBindingRepository is a mock implementation of outbound.LLMBindingRepository.
type MockLLMBindingRepository struct {
	mock.Mock
}

func (m *MockLLMBindingRepository) Create(ctx context.Context, binding *domain.LLMBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockLLMBindingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.LLMBinding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingRepository) GetByName(ctx context.Context, name string) (*domain.LLMBinding, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingRepository) List(ctx context.Context) ([]*domain.LLMBinding, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingRepository) Update(ctx context.Context, binding *domain.LLMBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockLLMBindingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestLLMBindingHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/llm-bindings", handler.Create)

		payload := map[string]interface{}{
			"name":                 "openai-gpt4o",
			"description":          "Production GPT-4o Binding",
			"provider":             "openai",
			"api_base_url":         "https://api.openai.com/v1",
			"api_key_secret_ref":    "openai-api-key",
			"default_model":       "gpt-4o",
			"default_temperature": 0.7,
			"default_max_tokens":   2048,
			"timeout_seconds":     30,
		}

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.LLMBinding")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/llm-bindings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "openai-gpt4o", respBody["name"])
		assert.NotEmpty(t, respBody["id"])
	})

	t.Run("Create LLM Binding Validation Failure", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/llm-bindings", handler.Create)

		// Missing name and invalid URL
		payload := map[string]interface{}{
			"provider":      "openai",
			"api_base_url":  "invalid-url",
			"default_model": "gpt-4o",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/llm-bindings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestLLMBindingHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get LLM Binding Found", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings/:id", handler.GetByID)

		id := uuid.New()
		binding := &domain.LLMBinding{
			ID:           id,
			Name:         "openai-gpt4o",
			Provider:     domain.ProviderOpenAI,
			APIBaseURL:   "https://api.openai.com/v1",
			DefaultModel: "gpt-4o",
		}

		repo.On("GetByID", mock.Anything, id).Return(binding, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/llm-bindings/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, id.String(), respBody["id"])
		assert.Equal(t, "openai-gpt4o", respBody["name"])
	})

	t.Run("Get LLM Binding Not Found", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings/:id", handler.GetByID)

		id := uuid.New()
		repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/llm-bindings/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestLLMBindingHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List LLM Bindings Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings", handler.List)

		bindings := []*domain.LLMBinding{
			{
				ID:           uuid.New(),
				Name:         "openai-gpt4o",
				Provider:     domain.ProviderOpenAI,
				APIBaseURL:   "https://api.openai.com/v1",
				DefaultModel: "gpt-4o",
			},
		}

		repo.On("List", mock.Anything).Return(bindings, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/llm-bindings", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody []map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Len(t, respBody, 1)
		assert.Equal(t, "openai-gpt4o", respBody[0]["name"])
	})
}

func TestLLMBindingHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/llm-bindings/:id", handler.Update)

		id := uuid.New()
		existing := &domain.LLMBinding{
			ID:                 id,
			Name:               "openai-gpt4o",
			Provider:           domain.ProviderOpenAI,
			APIBaseURL:         "https://api.openai.com/v1",
			APIKeySecretRef:    "openai-api-key",
			DefaultModel:       "gpt-4o",
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2048,
			TimeoutSeconds:     30,
			Status:             domain.StatusActive,
		}

		payload := map[string]interface{}{
			"name":                 "openai-gpt4o-updated",
			"description":          "Updated Binding Description",
			"provider":             "openai",
			"api_base_url":         "https://api.openai.com/v1",
			"api_key_secret_ref":    "openai-api-key-new",
			"default_model":       "gpt-4o",
			"default_temperature": 0.5,
			"default_max_tokens":   4096,
			"timeout_seconds":     60,
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.LLMBinding")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/llm-bindings/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "openai-gpt4o-updated", respBody["name"])
		assert.Equal(t, 0.5, respBody["default_temperature"])
	})
}

func TestLLMBindingHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingRepository)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/llm-bindings/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/llm-bindings/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}
