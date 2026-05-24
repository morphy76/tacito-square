package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type LLMBindingService struct {
	repo outbound.LLMBindingRepository
}

func NewLLMBindingService(repo outbound.LLMBindingRepository) *LLMBindingService {
	return &LLMBindingService{repo: repo}
}

func (s *LLMBindingService) Create(ctx context.Context, binding *model.LLMBinding) error {
	return s.repo.Create(ctx, binding)
}

func (s *LLMBindingService) GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *LLMBindingService) List(ctx context.Context) ([]*model.LLMBinding, error) {
	return s.repo.List(ctx)
}

func (s *LLMBindingService) Update(ctx context.Context, binding *model.LLMBinding) error {
	return s.repo.Update(ctx, binding)
}

func (s *LLMBindingService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
