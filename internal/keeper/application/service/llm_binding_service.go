package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

type LLMBindingService struct {
	repo  outbound.LLMBindingRepository
	cache sharedports.Cache
}

func NewLLMBindingService(repo outbound.LLMBindingRepository, cache sharedports.Cache) *LLMBindingService {
	return &LLMBindingService{repo: repo, cache: cache}
}

func (s *LLMBindingService) Create(ctx context.Context, binding *model.LLMBinding) error {
	return s.repo.Create(ctx, binding)
}

func (s *LLMBindingService) GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error) {
	var tenantID string
	if ten := tenant.FromContext(ctx); ten != nil {
		tenantID = ten.FullName()
	}

	cacheKey := fmt.Sprintf("%s:llm-bindings:%s", tenantID, id.String())

	if s.cache != nil && tenantID != "" {
		var cached model.LLMBinding
		err := s.cache.Get(ctx, cacheKey, &cached)
		if err == nil {
			return &cached, nil
		}
	}

	binding, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil && tenantID != "" {
		_ = s.cache.Set(ctx, cacheKey, binding, 300*time.Second)
	}

	return binding, nil
}

func (s *LLMBindingService) List(ctx context.Context) ([]*model.LLMBinding, error) {
	return s.repo.List(ctx)
}

func (s *LLMBindingService) Update(ctx context.Context, binding *model.LLMBinding) error {
	err := s.repo.Update(ctx, binding)
	if err != nil {
		return err
	}

	if s.cache != nil {
		var tenantID string
		if ten := tenant.FromContext(ctx); ten != nil {
			tenantID = ten.FullName()
		}
		if tenantID != "" {
			cacheKey := fmt.Sprintf("%s:llm-bindings:%s", tenantID, binding.ID.String())
			_ = s.cache.Invalidate(ctx, cacheKey)
		}
	}

	return nil
}

func (s *LLMBindingService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	if s.cache != nil {
		var tenantID string
		if ten := tenant.FromContext(ctx); ten != nil {
			tenantID = ten.FullName()
		}
		if tenantID != "" {
			cacheKey := fmt.Sprintf("%s:llm-bindings:%s", tenantID, id.String())
			_ = s.cache.Invalidate(ctx, cacheKey)
		}
	}

	return nil
}
