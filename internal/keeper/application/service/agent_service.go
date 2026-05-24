package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type AgentService struct {
	repo           outbound.AgentRepository
	crdCoordinator outbound.CRDCoordinator
}

func NewAgentService(repo outbound.AgentRepository, crdCoordinator outbound.CRDCoordinator) *AgentService {
	return &AgentService{
		repo:           repo,
		crdCoordinator: crdCoordinator,
	}
}

func (s *AgentService) Create(ctx context.Context, agent *model.Agent) error {
	return s.repo.Create(ctx, agent)
}

func (s *AgentService) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AgentService) List(ctx context.Context) ([]*model.Agent, error) {
	return s.repo.List(ctx)
}

func (s *AgentService) Update(ctx context.Context, agent *model.Agent) error {
	return s.repo.Update(ctx, agent)
}

func (s *AgentService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *AgentService) Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	// Outbound operations
	if err := s.repo.AssignToCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	return s.crdCoordinator.SubmitAgentCRD(ctx, agent)
}

func (s *AgentService) Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	agent, err := s.repo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}

	if err := s.repo.UnassignFromCommunity(ctx, agentID, communityID); err != nil {
		return err
	}

	return s.crdCoordinator.TeardownAgentCRD(ctx, agent)
}
