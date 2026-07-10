package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type CommunityService struct {
	repo           outbound.CommunityRepository
	assignmentRepo outbound.CommunityAssignmentRepository
}

func NewCommunityService(repo outbound.CommunityRepository, assignmentRepo outbound.CommunityAssignmentRepository) *CommunityService {
	return &CommunityService{
		repo:           repo,
		assignmentRepo: assignmentRepo,
	}
}

func (s *CommunityService) Create(ctx context.Context, community *model.Community) error {
	return s.repo.Create(ctx, community)
}

func (s *CommunityService) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CommunityService) List(ctx context.Context) ([]*model.Community, error) {
	return s.repo.List(ctx)
}

func (s *CommunityService) Update(ctx context.Context, community *model.Community) error {
	return s.repo.Update(ctx, community)
}

func (s *CommunityService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *CommunityService) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID, role model.AgentRole) error {
	community, err := s.repo.GetByID(ctx, communityID)
	if err != nil {
		return fmt.Errorf("get community by id: %w", err)
	}

	assignedRole, err := s.validateRole(ctx, community, role)
	if err != nil {
		return err
	}

	assignment := &model.CommunityAssignment{
		CommunityID: communityID,
		AgentID:     agentID,
		TenantID:    community.TenantID,
		Role:        assignedRole,
		AssignedAt:  time.Now().UTC(),
	}

	if err := assignment.Validate(); err != nil {
		return fmt.Errorf("validate assignment: %w", err)
	}

	return s.assignmentRepo.Create(ctx, assignment)
}

func (s *CommunityService) validateRole(ctx context.Context, community *model.Community, role model.AgentRole) (model.AgentRole, error) {
	switch community.Topology {
	case model.CommunityTopologySingleAgent:
		count, err := s.assignmentRepo.CountByCommunity(ctx, community.ID)
		if err != nil {
			return "", fmt.Errorf("count community assignments: %w", err)
		}
		if count >= 1 {
			return "", fmt.Errorf("community with single-agent topology cannot have more than one agent assigned")
		}
		return model.AgentRoleStandalone, nil
	case model.CommunityTopologyHubSpoke:
		if role != model.AgentRoleHub && role != model.AgentRoleSpoke {
			return "", fmt.Errorf("invalid role %s for hub-spoke topology", role)
		}
		if role == model.AgentRoleHub {
			hubs, err := s.assignmentRepo.CountHubs(ctx, community.ID)
			if err != nil {
				return "", fmt.Errorf("count hubs: %w", err)
			}
			if hubs >= 1 {
				return "", fmt.Errorf("community with hub-spoke topology cannot have more than one hub agent assigned")
			}
		}
		return role, nil
	default:
		return "", fmt.Errorf("unsupported topology: %s", community.Topology)
	}
}

func (s *CommunityService) Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	return s.assignmentRepo.Delete(ctx, communityID, agentID)
}

func (s *CommunityService) ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error) {
	return s.assignmentRepo.ListByCommunity(ctx, communityID)
}

