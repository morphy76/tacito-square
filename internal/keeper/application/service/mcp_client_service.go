package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type MCPClientService struct {
	repo outbound.MCPClientRepository
}

func NewMCPClientService(repo outbound.MCPClientRepository) *MCPClientService {
	return &MCPClientService{repo: repo}
}

func (s *MCPClientService) Create(ctx context.Context, client *model.MCPClient) error {
	return s.repo.Create(ctx, client)
}

func (s *MCPClientService) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPClient, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MCPClientService) List(ctx context.Context) ([]*model.MCPClient, error) {
	return s.repo.List(ctx)
}

func (s *MCPClientService) Update(ctx context.Context, client *model.MCPClient) error {
	return s.repo.Update(ctx, client)
}

func (s *MCPClientService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
