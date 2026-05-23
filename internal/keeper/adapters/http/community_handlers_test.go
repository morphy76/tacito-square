package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// MockCommunityRepository is a mock implementation of outbound.CommunityRepository.
type MockCommunityRepository struct {
	mock.Mock
}

func (m *MockCommunityRepository) Create(ctx context.Context, community *domain.Community) error {
	args := m.Called(ctx, community)
	return args.Error(0)
}

func (m *MockCommunityRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Community, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Community), args.Error(1)
}

func (m *MockCommunityRepository) GetByName(ctx context.Context, name string) (*domain.Community, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Community), args.Error(1)
}

func (m *MockCommunityRepository) List(ctx context.Context) ([]*domain.Community, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Community), args.Error(1)
}

func (m *MockCommunityRepository) Update(ctx context.Context, community *domain.Community) error {
	args := m.Called(ctx, community)
	return args.Error(0)
}

func (m *MockCommunityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCommunityHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Community Successfully", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities", handler.Create)

		payload := map[string]interface{}{
			"name":        "qa-community",
			"description": "QA Testing Community",
			"topology":    "hub-spoke",
			"configuration": map[string]interface{}{
				"max_messages_per_sec": float64(100),
			},
		}

		repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Community")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "qa-community", respBody["name"])
		assert.Equal(t, "hub-spoke", respBody["topology"])
		assert.NotEmpty(t, respBody["id"])
	})

	t.Run("Create Community Invalid Inputs", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities", handler.Create)

		payload := map[string]interface{}{
			"topology": "unsupported-topology",
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}

func TestCommunityHandlers_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Community Found", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:id", handler.GetByID)

		id := uuid.New()
		comm := &domain.Community{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "qa-community",
			Description: "QA testing community",
			Topology:    domain.CommunityTopologyHubSpoke,
			Status:      domain.CommunityStatusActive,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		repo.On("GetByID", mock.Anything, id).Return(comm, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody map[string]interface{}
		_ = json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.Equal(t, "qa-community", respBody["name"])
	})

	t.Run("Get Community Not Found", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:id", handler.GetByID)

		id := uuid.New()
		repo.On("GetByID", mock.Anything, id).Return(nil, errors.New("community not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestCommunityHandlers_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Communities Successfully", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities", handler.List)

		comms := []*domain.Community{
			{
				ID:       uuid.New(),
				Name:     "qa-community",
				Topology: domain.CommunityTopologyHubSpoke,
				Status:   domain.CommunityStatusCreated,
			},
		}

		repo.On("List", mock.Anything).Return(comms, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestCommunityHandlers_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Update Community Successfully", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/communities/:id", handler.Update)

		id := uuid.New()
		existing := &domain.Community{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "qa-community",
			Topology:    domain.CommunityTopologyHubSpoke,
			Status:      domain.CommunityStatusCreated,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		payload := map[string]interface{}{
			"name":        "qa-community-updated",
			"description": "Updated QA Community",
			"topology":    "hub-spoke",
			"status":      "active",
		}

		repo.On("GetByID", mock.Anything, id).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Community")).Return(nil)

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/communities/"+id.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestCommunityHandlers_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Delete Community Successfully", func(t *testing.T) {
		repo := new(MockCommunityRepository)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/communities/:id", handler.Delete)

		id := uuid.New()
		repo.On("Delete", mock.Anything, id).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/communities/"+id.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
}
