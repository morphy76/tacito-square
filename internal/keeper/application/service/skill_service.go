package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	domainsrv "github.com/morphy76/tacito-square/internal/keeper/domain/service"
)

type SkillService struct {
	repo      outbound.SkillRepository
	agentRepo outbound.AgentRepository
}

func NewSkillService(repo outbound.SkillRepository, agentRepo outbound.AgentRepository) *SkillService {
	return &SkillService{
		repo:      repo,
		agentRepo: agentRepo,
	}
}

func (s *SkillService) Create(ctx context.Context, skill *model.Skill) error {
	return s.repo.Create(ctx, skill)
}

func (s *SkillService) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SkillService) List(ctx context.Context) ([]*model.Skill, error) {
	return s.repo.List(ctx)
}

func (s *SkillService) Update(ctx context.Context, skill *model.Skill) error {
	return s.repo.Update(ctx, skill)
}

func (s *SkillService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *SkillService) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	return s.repo.AttachSkillToAgent(ctx, agentID, skillID)
}

func (s *SkillService) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	return s.repo.DetachSkillFromAgent(ctx, agentID, skillID)
}

func (s *SkillService) CreateCollection(ctx context.Context, collection *model.SkillCollection) error {
	return s.repo.CreateCollection(ctx, collection)
}

func (s *SkillService) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.SkillCollection, error) {
	return s.repo.GetCollectionByID(ctx, id)
}

func (s *SkillService) ListCollections(ctx context.Context) ([]*model.SkillCollection, error) {
	return s.repo.ListCollections(ctx)
}

func (s *SkillService) UpdateCollection(ctx context.Context, collection *model.SkillCollection) error {
	return s.repo.UpdateCollection(ctx, collection)
}

func (s *SkillService) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCollection(ctx, id)
}

func (s *SkillService) ResolveCollectionSkills(ctx context.Context, id uuid.UUID) ([]*model.Skill, error) {
	return s.repo.ResolveCollectionSkills(ctx, id)
}

func (s *SkillService) AttachCollectionToAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	return s.repo.AttachCollectionToAgent(ctx, agentID, collectionID)
}

func (s *SkillService) DetachCollectionFromAgent(ctx context.Context, agentID uuid.UUID, collectionID uuid.UUID) error {
	return s.repo.DetachCollectionFromAgent(ctx, agentID, collectionID)
}

func (s *SkillService) AddSkillToCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	return s.repo.AddSkillToCollection(ctx, collectionID, skillID)
}

func (s *SkillService) RemoveSkillFromCollection(ctx context.Context, collectionID uuid.UUID, skillID uuid.UUID) error {
	return s.repo.RemoveSkillFromCollection(ctx, collectionID, skillID)
}

func (s *SkillService) ResolveAgentSkills(ctx context.Context, agentID uuid.UUID) ([]*model.ResolvedSkill, error) {
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get agent for skill resolution: %w", err)
	}
	return domainsrv.ResolveAgentSkills(ctx, agent, s.repo)
}

func (s *SkillService) PatchStatus(ctx context.Context, id uuid.UUID, status model.SkillStatus) (*model.Skill, error) {
	if status != model.SkillStatusActive && status != model.SkillStatusSuspended && status != model.SkillStatusInactive {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	sk, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get skill for status patch: %w", err)
	}

	sk.Status = status
	sk.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, sk); err != nil {
		return nil, fmt.Errorf("update patched skill status: %w", err)
	}

	return sk, nil
}
