package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type SkillService struct {
	repo outbound.SkillRepository
}

func NewSkillService(repo outbound.SkillRepository) *SkillService {
	return &SkillService{repo: repo}
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
