package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type PromptService struct {
	repo outbound.PromptRepository
}

func NewPromptService(repo outbound.PromptRepository) *PromptService {
	return &PromptService{repo: repo}
}

func (s *PromptService) CreateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	return s.repo.CreateTemplate(ctx, template)
}

func (s *PromptService) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	return s.repo.GetTemplateByID(ctx, id)
}

func (s *PromptService) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *PromptService) UpdateTemplate(ctx context.Context, template *model.PromptTemplate) error {
	if model.IsSystemLocked(template.ID) {
		return fmt.Errorf("cannot update system-locked prompt template: %s", template.ID)
	}
	return s.repo.UpdateTemplate(ctx, template)
}

func (s *PromptService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if model.IsSystemLocked(id) {
		return fmt.Errorf("cannot delete system-locked prompt template: %s", id)
	}
	return s.repo.DeleteTemplate(ctx, id)
}

func (s *PromptService) CreateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return s.repo.CreateCollection(ctx, collection)
}

func (s *PromptService) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	return s.repo.GetCollectionByID(ctx, id)
}

func (s *PromptService) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	return s.repo.ListCollections(ctx)
}

func (s *PromptService) UpdateCollection(ctx context.Context, collection *model.PromptCollection) error {
	return s.repo.UpdateCollection(ctx, collection)
}

func (s *PromptService) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCollection(ctx, id)
}

func (s *PromptService) ResolveCollectionPrompts(ctx context.Context, id uuid.UUID) ([]*model.PromptTemplate, error) {
	return s.repo.ResolveCollectionPrompts(ctx, id)
}

func (s *PromptService) AddPromptToCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	return s.repo.AddPromptToCollection(ctx, collectionID, promptID)
}

func (s *PromptService) RemovePromptFromCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	return s.repo.RemovePromptFromCollection(ctx, collectionID, promptID)
}
