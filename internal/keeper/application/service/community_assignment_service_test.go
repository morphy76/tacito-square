package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockCommunityAssignmentRepository mocks the new outbound port.
type mockCommunityAssignmentRepository struct {
	mock.Mock
}

func (m *mockCommunityAssignmentRepository) Create(ctx context.Context, assignment *model.CommunityAssignment) error {
	return m.Called(ctx, assignment).Error(0)
}

func (m *mockCommunityAssignmentRepository) Delete(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	return m.Called(ctx, communityID, agentID).Error(0)
}

func (m *mockCommunityAssignmentRepository) ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error) {
	args := m.Called(ctx, communityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.CommunityAssignment), args.Error(1)
}

func (m *mockCommunityAssignmentRepository) CountHubs(ctx context.Context, communityID uuid.UUID) (int, error) {
	args := m.Called(ctx, communityID)
	return args.Int(0), args.Error(1)
}

func (m *mockCommunityAssignmentRepository) CountByCommunity(ctx context.Context, communityID uuid.UUID) (int, error) {
	args := m.Called(ctx, communityID)
	return args.Int(0), args.Error(1)
}

func TestCommunityService_Assign(t *testing.T) {
	ctx := context.Background()

	t.Run("Assign auto-sets standalone for single-agent topology, ignoring caller role", func(t *testing.T) {
		commRepo := new(mockCommunityRepository)
		assignRepo := new(mockCommunityAssignmentRepository)
		svc := NewCommunityService(commRepo, assignRepo)

		commID := uuid.New()
		agentID := uuid.New()

		community := &model.Community{
			ID:       commID,
			Name:     "single-comm",
			TenantID: "test-tenant.com",
			Topology: model.CommunityTopologySingleAgent,
			Status:   model.CommunityStatusActive,
		}

		commRepo.On("GetByID", ctx, commID).Return(community, nil)
		assignRepo.On("CountByCommunity", ctx, commID).Return(0, nil)

		// Assert that standalone role is used in the Create call
		assignRepo.On("Create", ctx, mock.MatchedBy(func(a *model.CommunityAssignment) bool {
			return a.CommunityID == commID && a.AgentID == agentID && a.Role == model.AgentRoleStandalone
		})).Return(nil)

		err := svc.Assign(ctx, commID, agentID, model.AgentRoleHub) // requested Hub, should be ignored/coerced to standalone
		assert.NoError(t, err)
		commRepo.AssertExpectations(t)
		assignRepo.AssertExpectations(t)
	})

	t.Run("Assign accepts hub for hub-spoke topology when hub count is 0", func(t *testing.T) {
		commRepo := new(mockCommunityRepository)
		assignRepo := new(mockCommunityAssignmentRepository)
		svc := NewCommunityService(commRepo, assignRepo)

		commID := uuid.New()
		agentID := uuid.New()

		community := &model.Community{
			ID:       commID,
			Name:     "hs-comm",
			TenantID: "test-tenant.com",
			Topology: model.CommunityTopologyHubSpoke,
			Status:   model.CommunityStatusActive,
		}

		commRepo.On("GetByID", ctx, commID).Return(community, nil)
		assignRepo.On("CountHubs", ctx, commID).Return(0, nil)
		assignRepo.On("Create", ctx, mock.MatchedBy(func(a *model.CommunityAssignment) bool {
			return a.CommunityID == commID && a.AgentID == agentID && a.Role == model.AgentRoleHub
		})).Return(nil)

		err := svc.Assign(ctx, commID, agentID, model.AgentRoleHub)
		assert.NoError(t, err)
	})

	t.Run("Assign rejects second hub for hub-spoke topology with domain conflict error", func(t *testing.T) {
		commRepo := new(mockCommunityRepository)
		assignRepo := new(mockCommunityAssignmentRepository)
		svc := NewCommunityService(commRepo, assignRepo)

		commID := uuid.New()
		agentID := uuid.New()

		community := &model.Community{
			ID:       commID,
			Name:     "hs-comm",
			TenantID: "test-tenant.com",
			Topology: model.CommunityTopologyHubSpoke,
			Status:   model.CommunityStatusActive,
		}

		commRepo.On("GetByID", ctx, commID).Return(community, nil)
		assignRepo.On("CountHubs", ctx, commID).Return(1, nil)

		err := svc.Assign(ctx, commID, agentID, model.AgentRoleHub)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "community with hub-spoke topology cannot have more than one hub agent assigned")
	})

	t.Run("Assign rejects hub role for single-agent topology with domain conflict error when coercion is not possible (wait, coercion always happens for single-agent topology, but let's test if coercion logic behaves correctly)", func(t *testing.T) {
		// If topology is single-agent, any role is forced to standalone.
		// What if we assign a second agent to a single-agent community?
		commRepo := new(mockCommunityRepository)
		assignRepo := new(mockCommunityAssignmentRepository)
		svc := NewCommunityService(commRepo, assignRepo)

		commID := uuid.New()
		agentID := uuid.New()

		community := &model.Community{
			ID:       commID,
			Name:     "single-comm",
			TenantID: "test-tenant.com",
			Topology: model.CommunityTopologySingleAgent,
			Status:   model.CommunityStatusActive,
		}

		commRepo.On("GetByID", ctx, commID).Return(community, nil)
		assignRepo.On("CountByCommunity", ctx, commID).Return(1, nil) // already has 1 agent

		err := svc.Assign(ctx, commID, agentID, model.AgentRoleStandalone)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "community with single-agent topology cannot have more than one agent assigned")
	})

	t.Run("Assign rejects standalone role for hub-spoke topology with domain conflict error", func(t *testing.T) {
		commRepo := new(mockCommunityRepository)
		assignRepo := new(mockCommunityAssignmentRepository)
		svc := NewCommunityService(commRepo, assignRepo)

		commID := uuid.New()
		agentID := uuid.New()

		community := &model.Community{
			ID:       commID,
			Name:     "hs-comm",
			TenantID: "test-tenant.com",
			Topology: model.CommunityTopologyHubSpoke,
			Status:   model.CommunityStatusActive,
		}

		commRepo.On("GetByID", ctx, commID).Return(community, nil)

		err := svc.Assign(ctx, commID, agentID, model.AgentRoleStandalone)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role standalone for hub-spoke topology")
	})
}

func TestCommunityService_ListByCommunity(t *testing.T) {
	ctx := context.Background()
	commRepo := new(mockCommunityRepository)
	assignRepo := new(mockCommunityAssignmentRepository)
	svc := NewCommunityService(commRepo, assignRepo)

	commID := uuid.New()
	expected := []*model.CommunityAssignment{
		{
			CommunityID: commID,
			AgentID:     uuid.New(),
			Role:        model.AgentRoleHub,
			AssignedAt:  time.Now(),
		},
	}

	assignRepo.On("ListByCommunity", ctx, commID).Return(expected, nil)

	res, err := svc.ListByCommunity(ctx, commID)
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestCommunityService_Unassign(t *testing.T) {
	ctx := context.Background()
	commRepo := new(mockCommunityRepository)
	assignRepo := new(mockCommunityAssignmentRepository)
	svc := NewCommunityService(commRepo, assignRepo)

	commID := uuid.New()
	agentID := uuid.New()

	assignRepo.On("Delete", ctx, commID, agentID).Return(nil)

	err := svc.Unassign(ctx, commID, agentID)
	assert.NoError(t, err)
}
