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
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
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

// MockLLMBindingUseCase is a mock implementation of inbound.LLMBindingUseCase.
type MockLLMBindingUseCase struct {
	mock.Mock
}

func (m *MockLLMBindingUseCase) Create(ctx context.Context, binding *model.LLMBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockLLMBindingUseCase) GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingUseCase) GetByName(ctx context.Context, name string) (*model.LLMBinding, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingUseCase) List(ctx context.Context) ([]*model.LLMBinding, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.LLMBinding), args.Error(1)
}

func (m *MockLLMBindingUseCase) Update(ctx context.Context, binding *model.LLMBinding) error {
	args := m.Called(ctx, binding)
	return args.Error(0)
}

func (m *MockLLMBindingUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestLLMBindingHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingUseCase)
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

		var capturedCtx context.Context
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.LLMBinding")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/llm-bindings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "null", resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Create LLM Binding Validation Failure", func(t *testing.T) {
		repo := new(MockLLMBindingUseCase)
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
		repo := new(MockLLMBindingUseCase)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings/:id", handler.GetByID)

		id := uuid.New()
		binding := &model.LLMBinding{
			ID:           id,
			Name:         "openai-gpt4o",
			Provider:     model.ProviderOpenAI,
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
		repo := new(MockLLMBindingUseCase)
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
		repo := new(MockLLMBindingUseCase)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings", handler.List)

		bindings := []*model.LLMBinding{
			{
				ID:           uuid.New(),
				Name:         "openai-gpt4o",
				Provider:     model.ProviderOpenAI,
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

	t.Run("List LLM Bindings Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockLLMBindingUseCase)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/llm-bindings", handler.List)

		repo.On("List", mock.Anything).Return(([]*model.LLMBinding)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/llm-bindings", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestLLMBindingHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingUseCase)
		handler := NewLLMBindingHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/llm-bindings/:id", handler.Update)

		id := uuid.New()
		existing := &model.LLMBinding{
			ID:                 id,
			Name:               "openai-gpt4o",
			Provider:           model.ProviderOpenAI,
			APIBaseURL:         "https://api.openai.com/v1",
			APIKeySecretRef:    "openai-api-key",
			DefaultModel:       "gpt-4o",
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2048,
			TimeoutSeconds:     30,
			Status:             model.StatusActive,
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
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.LLMBinding")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/llm-bindings/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "openai-gpt4o", respBody["name"])
		assert.Equal(t, 0.7, respBody["default_temperature"])
	})
}

func TestLLMBindingHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete LLM Binding Successfully", func(t *testing.T) {
		repo := new(MockLLMBindingUseCase)
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
