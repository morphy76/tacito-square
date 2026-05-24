package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAssignmentUseCase is a mock implementation of inbound.AssignmentUseCase.
type MockAssignmentUseCase struct {
	mock.Mock
}

func (m *MockAssignmentUseCase) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	args := m.Called(ctx, communityID, agentID)
	return args.Error(0)
}

func (m *MockAssignmentUseCase) Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	args := m.Called(ctx, communityID, agentID)
	return args.Error(0)
}

func TestAssignmentHandlers_Assign(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Assign Agent Successfully", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents/:agent_id", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "assigned")
	})

	t.Run("Assign Already Assigned Agent Returns 409", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents/:agent_id", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID).Return(errors.New("agent already assigned to community: ..."))

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.Contains(t, resp.Body.String(), "already assigned")
	})

	t.Run("Assign Missing Community Returns 404", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents/:agent_id", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID).Return(errors.New("community not found"))

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "not found")
	})

	t.Run("Assign Bad Parameters Returns 400", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents/:agent_id", handler.Assign)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/bad-uuid/agents/another-bad-uuid", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestAssignmentHandlers_Unassign(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Unassign Agent Successfully", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/communities/:community_id/agents/:agent_id", handler.Unassign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Unassign", mock.Anything, commID, agentID).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})

	t.Run("Unassign Not Assigned Agent Returns 400", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.DELETE("/api/v1/communities/:community_id/agents/:agent_id", handler.Unassign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Unassign", mock.Anything, commID, agentID).Return(errors.New("agent is not assigned to community: ..."))

		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/communities/"+commID.String()+"/agents/"+agentID.String(), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "not assigned")
	})
}
