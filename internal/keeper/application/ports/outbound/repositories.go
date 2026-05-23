package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
)

// LLMBindingRepository defines the persistent storage operations for LLM Provider Bindings.
type LLMBindingRepository interface {
	Create(ctx context.Context, binding *domain.LLMBinding) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.LLMBinding, error)
	GetByName(ctx context.Context, name string) (*domain.LLMBinding, error)
	List(ctx context.Context) ([]*domain.LLMBinding, error)
	Update(ctx context.Context, binding *domain.LLMBinding) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MCPServerRepository defines the persistent storage operations for MCP Server configurations.
type MCPServerRepository interface {
	Create(ctx context.Context, server *domain.MCPServer) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPServer, error)
	GetByName(ctx context.Context, name string) (*domain.MCPServer, error)
	List(ctx context.Context) ([]*domain.MCPServer, error)
	Update(ctx context.Context, server *domain.MCPServer) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SkillRepository defines the persistent storage operations for Skill Collections.
type SkillRepository interface {
	Create(ctx context.Context, skill *domain.Skill) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Skill, error)
	GetByName(ctx context.Context, name string) (*domain.Skill, error)
	List(ctx context.Context) ([]*domain.Skill, error)
	Update(ctx context.Context, skill *domain.Skill) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Agent-Skill associations
	AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
	DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
	ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Skill, error)
}
