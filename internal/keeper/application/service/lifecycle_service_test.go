package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)


type mockAgentRepository struct {
	mock.Mock
}

func (m *mockAgentRepository) Create(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) List(ctx context.Context) ([]*model.Agent, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Agent), args.Error(1)
}
func (m *mockAgentRepository) Update(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockAgentRepository) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return m.Called(ctx, agentID, communityID).Error(0)
}
func (m *mockAgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return m.Called(ctx, agentID, communityID).Error(0)
}

type mockCommunityRepository struct {
	mock.Mock
}

func (m *mockCommunityRepository) Create(ctx context.Context, community *model.Community) error {
	return m.Called(ctx, community).Error(0)
}
func (m *mockCommunityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) GetByName(ctx context.Context, name string) (*model.Community, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) List(ctx context.Context) ([]*model.Community, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Community), args.Error(1)
}
func (m *mockCommunityRepository) Update(ctx context.Context, community *model.Community) error {
	return m.Called(ctx, community).Error(0)
}
func (m *mockCommunityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

type mockCRDCoordinator struct {
	mock.Mock
}

func (m *mockCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockCRDCoordinator) TeardownAgentCRD(ctx context.Context, agent *model.Agent) error {
	return m.Called(ctx, agent).Error(0)
}
func (m *mockCRDCoordinator) GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1alpha1.TacitoAgentStatus), args.Error(1)
}

func TestLifecycleService_DeployAgent_Success(t *testing.T) {
	agentRepo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	crdCoord := new(mockCRDCoordinator)

	svc := service.NewLifecycleService(agentRepo, commRepo, crdCoord, nil)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    ten.FullName(),
		Name:        "test-agent",
		CommunityID: &commID,
		Status:      model.AgentStatusStopped,
	}

	agentRepo.On("GetByID", mock.Anything, agentID).Return(agent, nil)
	crdCoord.On("SubmitAgentCRD", mock.Anything, agent).Return(nil)
	agentRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.Status == model.AgentStatusPending
	})).Return(nil)

	err := svc.DeployAgent(ctx, agentID)
	assert.NoError(t, err)

	agentRepo.AssertExpectations(t)
	crdCoord.AssertExpectations(t)
}

func TestLifecycleService_DeployAgent_UnassignedError(t *testing.T) {
	agentRepo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	crdCoord := new(mockCRDCoordinator)

	svc := service.NewLifecycleService(agentRepo, commRepo, crdCoord, nil)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    ten.FullName(),
		Name:        "test-agent",
		CommunityID: nil,
		Status:      model.AgentStatusDefined,
	}

	agentRepo.On("GetByID", mock.Anything, agentID).Return(agent, nil)

	err := svc.DeployAgent(ctx, agentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be assigned to a community")
}

func TestLifecycleService_DeployAgent_TenantMismatch(t *testing.T) {
	agentRepo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	crdCoord := new(mockCRDCoordinator)

	svc := service.NewLifecycleService(agentRepo, commRepo, crdCoord, nil)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	agentID := uuid.New()
	commID := uuid.New()
	agent := &model.Agent{
		ID:          agentID,
		TenantID:    "different-tenant.com",
		Name:        "test-agent",
		CommunityID: &commID,
		Status:      model.AgentStatusStopped,
	}

	agentRepo.On("GetByID", mock.Anything, agentID).Return(agent, nil)

	err := svc.DeployAgent(ctx, agentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found") // returns 404/not found equivalent
}

func TestLifecycleService_DeployCommunity_MultiStatusPartialSuccess(t *testing.T) {
	agentRepo := new(mockAgentRepository)
	commRepo := new(mockCommunityRepository)
	crdCoord := new(mockCRDCoordinator)

	svc := service.NewLifecycleService(agentRepo, commRepo, crdCoord, nil)

	ten, _ := tenant.New("acme.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	commID := uuid.New()
	comm := &model.Community{
		ID:       commID,
		TenantID: ten.FullName(),
		Name:     "test-comm",
		Status:   model.CommunityStatusCreated,
	}

	agentID1 := uuid.New()
	agentID2 := uuid.New()
	agents := []*model.Agent{
		{
			ID:          agentID1,
			TenantID:    ten.FullName(),
			Name:        "agent-1",
			CommunityID: &commID,
			Status:      model.AgentStatusStopped,
		},
		{
			ID:          agentID2,
			TenantID:    ten.FullName(),
			Name:        "agent-2",
			CommunityID: &commID,
			Status:      model.AgentStatusStopped,
		},
	}

	commRepo.On("GetByID", mock.Anything, commID).Return(comm, nil)
	agentRepo.On("List", mock.Anything).Return(agents, nil)

	// mock DeployAgent behaviours: first succeeds, second fails
	agentRepo.On("GetByID", mock.Anything, agentID1).Return(agents[0], nil)
	crdCoord.On("SubmitAgentCRD", mock.Anything, agents[0]).Return(nil)
	agentRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.ID == agentID1 && a.Status == model.AgentStatusPending
	})).Return(nil)

	agentRepo.On("GetByID", mock.Anything, agentID2).Return(agents[1], nil)
	agentRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Agent) bool {
		return a.ID == agentID2 && a.Status == model.AgentStatusPending
	})).Return(nil)
	crdCoord.On("SubmitAgentCRD", mock.Anything, agents[1]).Return(errors.New("k8s API error"))

	// We still update the community to active if at least one starts, or let's say community status transitions to active
	commRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *model.Community) bool {
		return c.Status == model.CommunityStatusActive
	})).Return(nil)

	res, err := svc.DeployCommunity(ctx, commID)
	assert.NoError(t, err) // service method returns result breakdown, not overall failure
	require.NotNil(t, res)
	assert.Equal(t, commID, res.CommunityID)
	assert.Equal(t, "partial_success", res.Status)
	assert.Len(t, res.Agents, 2)

	var successID, failID uuid.UUID
	for _, a := range res.Agents {
		if a.Status == "deployed" {
			successID = a.AgentID
		} else if a.Status == "failed" {
			failID = a.AgentID
			assert.Contains(t, a.Error, "k8s API error")
		}
	}
	assert.Equal(t, agentID1, successID)
	assert.Equal(t, agentID2, failID)
}
