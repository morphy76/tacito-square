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

// MockMCPServerUseCase is a mock implementation of inbound.MCPServerUseCase.
type MockMCPServerUseCase struct {
	mock.Mock
}

func (m *MockMCPServerUseCase) Create(ctx context.Context, server *model.MCPServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockMCPServerUseCase) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPServer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MCPServer), args.Error(1)
}

func (m *MockMCPServerUseCase) GetByName(ctx context.Context, name string) (*model.MCPServer, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.MCPServer), args.Error(1)
}

func (m *MockMCPServerUseCase) List(ctx context.Context) ([]*model.MCPServer, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.MCPServer), args.Error(1)
}

func (m *MockMCPServerUseCase) Update(ctx context.Context, server *model.MCPServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockMCPServerUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMCPServerHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Stdio MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-servers", handler.Create)

		payload := map[string]interface{}{
			"name":        "sqlite-mcp",
			"description": "Local SQLite MCP",
			"transport":   "stdio",
			"command":     "mcp-sqlite",
			"args":        []string{"--db", "test.db"},
			"env":         map[string]string{"DEBUG": "true"},
		}

		var capturedCtx context.Context
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MCPServer")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "null", resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Create SSE MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-servers", handler.Create)

		payload := map[string]interface{}{
			"name":        "github-mcp",
			"description": "Remote GitHub MCP",
			"transport":   "sse",
			"url":         "https://mcp.github.com/events",
		}

		var capturedCtx context.Context
		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.MCPServer")).Return(nil).Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
		})

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Equal(t, "null", resp.Body.String())
		if assert.NotNil(t, capturedCtx) {
			assert.ErrorIs(t, capturedCtx.Err(), context.Canceled)
		}
	})

	t.Run("Create MCP Server Transport Invariant Failure (SSE missing URL)", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/mcp-servers", handler.Create)

		payload := map[string]interface{}{
			"name":      "github-mcp",
			"transport": "sse",
			// url is missing
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestMCPServerHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get MCP Server Found", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-servers/:id", handler.GetByID)

		id := uuid.New()
		server := &model.MCPServer{
			ID:        id,
			Name:      "github-mcp",
			Transport: model.TransportSSE,
			URL:       "https://mcp.github.com/events",
			Status:    model.MCPServerStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		repo.On("GetByID", mock.Anything, id).Return(server, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-servers/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		_ = json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.Equal(t, "github-mcp", respBody["name"])
	})
}

func TestMCPServerHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List MCP Servers Successfully", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-servers", handler.List)

		servers := []*model.MCPServer{
			{
				ID:        uuid.New(),
				Name:      "sqlite-mcp",
				Transport: model.TransportStdio,
				Command:   "sqlite-mcp",
				Status:    model.MCPServerStatusActive,
			},
		}

		repo.On("List", mock.Anything).Return(servers, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-servers", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("List MCP Servers Returns Empty Array When Nil", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-servers", handler.List)

		repo.On("List", mock.Anything).Return(([]*model.MCPServer)(nil), nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-servers", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "[]", resp.Body.String())
	})
}

func TestMCPServerHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update MCP Server Returns Previous Unmodified State", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/mcp-servers/:id", handler.Update)

		id := uuid.New()
		existing := &model.MCPServer{
			ID:        id,
			TenantID:  "test-tenant.com",
			Name:      "sqlite-mcp",
			Transport: model.TransportStdio,
			Command:   "sqlite-mcp",
			Status:    model.MCPServerStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "sqlite-mcp-updated",
			"description": "Updated SQLite MCP",
			"transport":   "stdio",
			"command":     "sqlite-mcp-v2",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.MCPServer")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/mcp-servers/"+id.String(), bytes.NewBuffer(body))
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

func TestMCPServerHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerUseCase)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/mcp-servers/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/mcp-servers/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}
