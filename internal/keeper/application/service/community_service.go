package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type CommunityService struct {
	repo outbound.CommunityRepository
}

func NewCommunityService(repo outbound.CommunityRepository) *CommunityService {
	return &CommunityService{repo: repo}
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
