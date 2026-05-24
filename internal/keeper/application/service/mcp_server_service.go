package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

type MCPServerService struct {
	repo outbound.MCPServerRepository
}

func NewMCPServerService(repo outbound.MCPServerRepository) *MCPServerService {
	return &MCPServerService{repo: repo}
}

func (s *MCPServerService) Create(ctx context.Context, server *model.MCPServer) error {
	return s.repo.Create(ctx, server)
}

func (s *MCPServerService) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPServer, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MCPServerService) List(ctx context.Context) ([]*model.MCPServer, error) {
	return s.repo.List(ctx)
}

func (s *MCPServerService) Update(ctx context.Context, server *model.MCPServer) error {
	return s.repo.Update(ctx, server)
}

func (s *MCPServerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
