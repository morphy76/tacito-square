package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLifecycleUseCase struct {
	mock.Mock
}

func (m *MockLifecycleUseCase) DeployAgent(ctx context.Context, agentID uuid.UUID) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

func (m *MockLifecycleUseCase) UndeployAgent(ctx context.Context, agentID uuid.UUID) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

func (m *MockLifecycleUseCase) GetAgentStatus(ctx context.Context, agentID uuid.UUID) (*inbound.AgentStatusDetails, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.AgentStatusDetails), args.Error(1)
}

func (m *MockLifecycleUseCase) DeployCommunity(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityDeploymentDetails, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.CommunityDeploymentDetails), args.Error(1)
}

func (m *MockLifecycleUseCase) UndeployCommunity(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityDeploymentDetails, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.CommunityDeploymentDetails), args.Error(1)
}

func (m *MockLifecycleUseCase) GetCommunityStatus(ctx context.Context, communityID uuid.UUID) (*inbound.CommunityStatusDetails, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*inbound.CommunityStatusDetails), args.Error(1)
}

func TestLifecycleHandlers_DeployAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Deploy Agent Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/deploy", handler.DeployAgent)

		agentID := uuid.New()
		uc.On("DeployAgent", mock.Anything, agentID).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/deploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
		uc.AssertExpectations(t)
	})

	t.Run("Deploy Agent Unassigned Error", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/deploy", handler.DeployAgent)

		agentID := uuid.New()
		uc.On("DeployAgent", mock.Anything, agentID).Return(errors.New("agent must be assigned to a community before deploying"))

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/deploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
		uc.AssertExpectations(t)
	})

	t.Run("Deploy Agent Conflict (Already pending/running)", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/deploy", handler.DeployAgent)

		agentID := uuid.New()
		uc.On("DeployAgent", mock.Anything, agentID).Return(errors.New("agent is already pending or running"))

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/deploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
		uc.AssertExpectations(t)
	})

	t.Run("Deploy Agent Not Found / Tenant Mismatch", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/deploy", handler.DeployAgent)

		agentID := uuid.New()
		uc.On("DeployAgent", mock.Anything, agentID).Return(errors.New("agent not found: " + agentID.String()))

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/deploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
		uc.AssertExpectations(t)
	})
}

func TestLifecycleHandlers_UndeployAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Undeploy Agent Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/undeploy", handler.UndeployAgent)

		agentID := uuid.New()
		uc.On("UndeployAgent", mock.Anything, agentID).Return(nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/undeploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		uc.AssertExpectations(t)
	})

	t.Run("Undeploy Agent Conflict (Already stopped)", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/agents/:agent_id/undeploy", handler.UndeployAgent)

		agentID := uuid.New()
		uc.On("UndeployAgent", mock.Anything, agentID).Return(errors.New("agent is already undeployed/stopped"))

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/undeploy", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
		uc.AssertExpectations(t)
	})
}

func TestLifecycleHandlers_GetAgentStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Agent Status Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/agents/:agent_id/status", handler.GetAgentStatus)

		agentID := uuid.New()
		details := &inbound.AgentStatusDetails{
			AgentID:   agentID,
			Status:    model.AgentStatusRunning,
			Message:   "Pod healthy and running",
			Replicas:  1,
			UpdatedAt: time.Now().UTC(),
		}
		uc.On("GetAgentStatus", mock.Anything, agentID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/agents/%s/status", agentID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		
		var out inbound.AgentStatusDetails
		err := json.Unmarshal(resp.Body.Bytes(), &out)
		assert.NoError(t, err)
		assert.Equal(t, agentID, out.AgentID)
		assert.Equal(t, model.AgentStatusRunning, out.Status)
		assert.Equal(t, "Pod healthy and running", out.Message)
		uc.AssertExpectations(t)
	})
}

func TestLifecycleHandlers_DeployCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Deploy Community Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/deploy", handler.DeployCommunity)

		commID := uuid.New()
		details := &inbound.CommunityDeploymentDetails{
			CommunityID: commID,
			Status:      "success",
			Agents:      []inbound.AgentDeploymentResult{},
		}
		uc.On("DeployCommunity", mock.Anything, commID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/deploy", commID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusAccepted, resp.Code)
		uc.AssertExpectations(t)
	})

	t.Run("Deploy Community Partial Success (207 Multi-Status)", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/deploy", handler.DeployCommunity)

		commID := uuid.New()
		details := &inbound.CommunityDeploymentDetails{
			CommunityID: commID,
			Status:      "partial_success",
			Agents: []inbound.AgentDeploymentResult{
				{AgentID: uuid.New(), Status: "deployed"},
				{AgentID: uuid.New(), Status: "failed", Error: "k8s API error"},
			},
		}
		uc.On("DeployCommunity", mock.Anything, commID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/deploy", commID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusMultiStatus, resp.Code)
		
		var out inbound.CommunityDeploymentDetails
		err := json.Unmarshal(resp.Body.Bytes(), &out)
		assert.NoError(t, err)
		assert.Equal(t, "partial_success", out.Status)
		assert.Len(t, out.Agents, 2)
		uc.AssertExpectations(t)
	})
}

func TestLifecycleHandlers_UndeployCommunity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Undeploy Community Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/undeploy", handler.UndeployCommunity)

		commID := uuid.New()
		details := &inbound.CommunityDeploymentDetails{
			CommunityID: commID,
			Status:      "success",
			Agents:      []inbound.AgentDeploymentResult{},
		}
		uc.On("UndeployCommunity", mock.Anything, commID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/undeploy", commID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		uc.AssertExpectations(t)
	})

	t.Run("Undeploy Community Partial Success (207 Multi-Status)", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.POST("/api/v1/communities/:community_id/undeploy", handler.UndeployCommunity)

		commID := uuid.New()
		details := &inbound.CommunityDeploymentDetails{
			CommunityID: commID,
			Status:      "partial_success",
			Agents: []inbound.AgentDeploymentResult{
				{AgentID: uuid.New(), Status: "stopped"},
				{AgentID: uuid.New(), Status: "failed", Error: "k8s connection refused"},
			},
		}
		uc.On("UndeployCommunity", mock.Anything, commID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/communities/%s/undeploy", commID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusMultiStatus, resp.Code)
		
		var out inbound.CommunityDeploymentDetails
		err := json.Unmarshal(resp.Body.Bytes(), &out)
		assert.NoError(t, err)
		assert.Equal(t, "partial_success", out.Status)
		assert.Len(t, out.Agents, 2)
		uc.AssertExpectations(t)
	})
}

func TestLifecycleHandlers_GetCommunityStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Get Community Status Successfully", func(t *testing.T) {
		uc := new(MockLifecycleUseCase)
		handler := NewLifecycleHandler(uc)

		r := gin.New()
		r.Use(testTenantMiddleware())
		r.GET("/api/v1/communities/:community_id/status", handler.GetCommunityStatus)

		commID := uuid.New()
		details := &inbound.CommunityStatusDetails{
			CommunityID: commID,
			Status:      model.CommunityStatusActive,
			Agents: []inbound.AgentStatusDetails{
				{AgentID: uuid.New(), Status: model.AgentStatusRunning, Message: "Pod healthy"},
			},
		}
		uc.On("GetCommunityStatus", mock.Anything, commID).Return(details, nil)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/communities/%s/status", commID), nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		
		var out inbound.CommunityStatusDetails
		err := json.Unmarshal(resp.Body.Bytes(), &out)
		assert.NoError(t, err)
		assert.Equal(t, commID, out.CommunityID)
		assert.Equal(t, model.CommunityStatusActive, out.Status)
		assert.Len(t, out.Agents, 1)
		uc.AssertExpectations(t)
	})
}
