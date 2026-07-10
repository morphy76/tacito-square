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

// MockAssignmentUseCase is a mock implementation of inbound.AssignmentUseCase.
type MockAssignmentUseCase struct {
	mock.Mock
}

func (m *MockAssignmentUseCase) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID, role model.AgentRole) error {
	args := m.Called(ctx, communityID, agentID, role)
	return args.Error(0)
}

func (m *MockAssignmentUseCase) Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	args := m.Called(ctx, communityID, agentID)
	return args.Error(0)
}

func (m *MockAssignmentUseCase) ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CommunityAssignment), args.Error(1)
}

func TestAssignmentHandlers_Assign(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Assign Agent Successfully", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID, model.AgentRoleHub).Return(nil)

		payload := map[string]interface{}{
			"agent_id": agentID.String(),
			"role":     "hub",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var respBody map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, agentID.String(), respBody["agent_id"])
		assert.Equal(t, "hub", respBody["role"])
		assert.Contains(t, respBody, "assigned_at")
	})

	t.Run("Assign Already Assigned Agent Returns 409", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID, model.AgentRoleHub).Return(errors.New("agent already assigned to community"))

		payload := map[string]interface{}{
			"agent_id": agentID.String(),
			"role":     "hub",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.Contains(t, resp.Body.String(), "already assigned")
	})

	t.Run("Assign Hub-Conflict Returns 409", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents", handler.Assign)

		commID := uuid.New()
		agentID := uuid.New()

		usecase.On("Assign", mock.Anything, commID, agentID, model.AgentRoleHub).Return(errors.New("community with hub-spoke topology cannot have more than one hub agent assigned"))

		payload := map[string]interface{}{
			"agent_id": agentID.String(),
			"role":     "hub",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.Contains(t, resp.Body.String(), "cannot have more than one hub agent")
	})

	t.Run("Assign Bad Parameters (Missing agent_id) Returns 400", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/agents", handler.Assign)

		commID := uuid.New()

		payload := map[string]interface{}{
			"role": "hub",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/communities/"+commID.String()+"/agents", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestAssignmentHandlers_ListAssignments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("List Assignments Successfully", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/agents", handler.ListAssignments)

		commID := uuid.New()
		agentID := uuid.New()
		assignedAt := time.Now().UTC()

		expected := []*model.CommunityAssignment{
			{
				CommunityID: commID,
				AgentID:     agentID,
				Role:        model.AgentRoleHub,
				AssignedAt:  assignedAt,
			},
		}

		usecase.On("ListByCommunity", mock.Anything, commID).Return(expected, nil)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var respBody []map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Len(t, respBody, 1)
		assert.Equal(t, agentID.String(), respBody[0]["agent_id"])
		assert.Equal(t, "hub", respBody[0]["role"])
	})

	t.Run("List Assignments Not Found Community", func(t *testing.T) {
		usecase := new(MockAssignmentUseCase)
		handler := NewAssignmentHandler(usecase)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/agents", handler.ListAssignments)

		commID := uuid.New()

		usecase.On("ListByCommunity", mock.Anything, commID).Return(([]*model.CommunityAssignment)(nil), errors.New("community not found"))

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/communities/"+commID.String()+"/agents", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "not found")
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
}
