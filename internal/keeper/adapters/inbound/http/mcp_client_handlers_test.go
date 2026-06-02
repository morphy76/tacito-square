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

// MockMCPClientUseCase is a mock implementation of inbound.MCPClientUseCase.
type MockMCPClientUseCase struct {
	mock.Mock
}

func (m *MockMCPClientUseCase) Create(ctx context.Context, client *model.MCPClient) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *MockMCPClientUseCase) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPClient, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MCPClient), args.Error(1)
}

func (m *MockMCPClientUseCase) GetByName(ctx context.Context, name string) (*model.MCPClient, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MCPClient), args.Error(1)
}

func (m *MockMCPClientUseCase) List(ctx context.Context) ([]*model.MCPClient, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MCPClient), args.Error(1)
}

func (m *MockMCPClientUseCase) Update(ctx context.Context, client *model.MCPClient) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *MockMCPClientUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMCPClientHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Stdio MCP Client Successfully", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-clients", handler.Create)

		payload := map[string]interface{}{
			"name":        "sqlite-mcp",
			"description": "Local SQLite MCP",
			"transport":   "stdio",
			"command":     "mcp-sqlite",
			"args":        []string{"--db", "test.db"},
			"env":         map[string]string{"DEBUG": "true"},
		}

		var capturedCtx context.Context
		var capturedClientID uuid.UUID
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MCPClient")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedClientID = args.Get(1).(*model.MCPClient).ID
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "/api/v1/mcp-clients/"+capturedClientID.String(), resp.Header().Get("Location"))
		assert.Empty(t, resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Create SSE MCP Client Successfully", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-clients", handler.Create)

		payload := map[string]interface{}{
			"name":        "github-mcp",
			"description": "Remote GitHub MCP",
			"transport":   "sse",
			"url":         "https://mcp.github.com/events",
		}

		var capturedCtx context.Context
		var capturedClientID uuid.UUID
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MCPClient")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedClientID = args.Get(1).(*model.MCPClient).ID
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "/api/v1/mcp-clients/"+capturedClientID.String(), resp.Header().Get("Location"))
		assert.Empty(t, resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Create MCP Client Transport Invariant Failure (SSE missing URL)", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-clients", handler.Create)

		payload := map[string]interface{}{
			"name":      "github-mcp",
			"transport": "sse",
			// url is missing
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-clients", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestMCPClientHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get MCP Client Found", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-clients/:id", handler.GetByID)

		id := uuid.New()
		clientObj := &model.MCPClient{
			ID:        id,
			Name:      "github-mcp",
			Transport: model.TransportSSE,
			URL:       "https://mcp.github.com/events",
			Status:    model.MCPClientStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		repo.On("GetByID", mock.Anything, id).Return(clientObj, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-clients/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		_ = json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.Equal(t, "github-mcp", respBody["name"])
	})
}

func TestMCPClientHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List MCP Clients Successfully", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-clients", handler.List)

		clients := []*model.MCPClient{
			{
				ID:        uuid.New(),
				Name:      "sqlite-mcp",
				Transport: model.TransportStdio,
				Command:   "sqlite-mcp",
				Status:    model.MCPClientStatusActive,
			},
		}

		repo.On("List", mock.Anything).Return(clients, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-clients", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("List MCP Clients Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-clients", handler.List)

		repo.On("List", mock.Anything).Return(([]*model.MCPClient)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-clients", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestMCPClientHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update MCP Client Returns Previous Unmodified State", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/mcp-clients/:id", handler.Update)

		id := uuid.New()
		existing := &model.MCPClient{
			ID:        id,
			TenantID:  "test-tenant.com",
			Name:      "sqlite-mcp",
			Transport: model.TransportStdio,
			Command:   "sqlite-mcp",
			Status:    model.MCPClientStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "sqlite-mcp-updated",
			"description": "Updated SQLite MCP",
			"transport":   "stdio",
			"command":     "sqlite-mcp-v2",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MCPClient")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/mcp-clients/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "sqlite-mcp", respBody["name"])
		assert.Equal(t, "stdio", respBody["transport"])
	})
}

func TestMCPClientHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete MCP Client Successfully", func(t *testing.T) {
		repo := new(MockMCPClientUseCase)
		handler := NewMCPClientHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/mcp-clients/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/mcp-clients/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}
