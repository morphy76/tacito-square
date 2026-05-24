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
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommunityUseCase is a mock implementation of inbound.CommunityUseCase.
type MockCommunityUseCase struct {
	mock.Mock
}

func (m *MockCommunityUseCase) Create(ctx context.Context, community *model.Community) error {
	args := m.Called(ctx, community)
	return args.Error(0)
}

func (m *MockCommunityUseCase) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}

func (m *MockCommunityUseCase) GetByName(ctx context.Context, name string) (*model.Community, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}

func (m *MockCommunityUseCase) List(ctx context.Context) ([]*model.Community, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Community), args.Error(1)
}

func (m *MockCommunityUseCase) Update(ctx context.Context, community *model.Community) error {
	args := m.Called(ctx, community)
	return args.Error(0)
}

func (m *MockCommunityUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestCommunityHandlers_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Create Community Successfully", func(t *testing.T) {
		repo := new(MockCommunityUseCase)
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

		repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Community")).Return(nil)

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
		repo := new(MockCommunityUseCase)
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
		repo := new(MockCommunityUseCase)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:id", handler.GetByID)

		id := uuid.New()
		comm := &model.Community{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "qa-community",
			Description: "QA testing community",
			Topology:    model.CommunityTopologyHubSpoke,
			Status:      model.CommunityStatusActive,
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
		repo := new(MockCommunityUseCase)
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
		repo := new(MockCommunityUseCase)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities", handler.List)

		comms := []*model.Community{
			{
				ID:       uuid.New(),
				Name:     "qa-community",
				Topology: model.CommunityTopologyHubSpoke,
				Status:   model.CommunityStatusCreated,
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
		repo := new(MockCommunityUseCase)
		handler := NewCommunityHandler(repo)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.PUT("/api/v1/communities/:id", handler.Update)

		id := uuid.New()
		existing := &model.Community{
			ID:          id,
			TenantID:    "test-tenant.com",
			Name:        "qa-community",
			Topology:    model.CommunityTopologyHubSpoke,
			Status:      model.CommunityStatusCreated,
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
		repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Community")).Return(nil)

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
		repo := new(MockCommunityUseCase)
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
