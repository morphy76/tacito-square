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

// MockMCPServerRepository is a mock implementation of outbound.MCPServerRepository.
type MockMCPServerRepository struct {
	mock.Mock
}

func (m *MockMCPServerRepository) Create(ctx context.Context, server *domain.MCPServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockMCPServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPServer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MCPServer), args.Error(1)
}

func (m *MockMCPServerRepository) GetByName(ctx context.Context, name string) (*domain.MCPServer, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MCPServer), args.Error(1)
}

func (m *MockMCPServerRepository) List(ctx context.Context) ([]*domain.MCPServer, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.MCPServer), args.Error(1)
}

func (m *MockMCPServerRepository) Update(ctx context.Context, server *domain.MCPServer) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockMCPServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestMCPServerHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Stdio MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerRepository)
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

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.MCPServer")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "sqlite-mcp", respBody["name"])
		assert.Equal(t, "stdio", respBody["transport"])
		assert.NotEmpty(t, respBody["id"])
	})

	t.Run("Create SSE MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerRepository)
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

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.MCPServer")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/mcp-servers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
	})

	t.Run("Create MCP Server Transport Invariant Failure (SSE missing URL)", func(t *testing.T) {
		repo := new(MockMCPServerRepository)
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
		repo := new(MockMCPServerRepository)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-servers/:id", handler.GetByID)

		id := uuid.New()
		server := &domain.MCPServer{
			ID:        id,
			Name:      "github-mcp",
			Transport: domain.TransportSSE,
			URL:       "https://mcp.github.com/events",
			Status:    domain.MCPServerStatusActive,
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
		repo := new(MockMCPServerRepository)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/mcp-servers", handler.List)

		servers := []*domain.MCPServer{
			{
				ID:        uuid.New(),
				Name:      "sqlite-mcp",
				Transport: domain.TransportStdio,
				Command:   "sqlite-mcp",
				Status:    domain.MCPServerStatusActive,
			},
		}

		repo.On("List", mock.Anything).Return(servers, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/mcp-servers", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestMCPServerHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerRepository)
		handler := NewMCPServerHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/mcp-servers/:id", handler.Update)

		id := uuid.New()
		existing := &domain.MCPServer{
			ID:        id,
			Name:      "sqlite-mcp",
			Transport: domain.TransportStdio,
			Command:   "sqlite-mcp",
			Status:    domain.MCPServerStatusActive,
		}

		payload := map[string]interface{}{
			"name":        "sqlite-mcp-updated",
			"description": "Updated SQLite MCP",
			"transport":   "stdio",
			"command":     "sqlite-mcp-v2",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.MCPServer")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/mcp-servers/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestMCPServerHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete MCP Server Successfully", func(t *testing.T) {
		repo := new(MockMCPServerRepository)
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
