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

// PromptRepository defines the persistent storage operations for Prompt Templates and Prompt Collections.
type PromptRepository interface {
	// PromptTemplate CRUD
	CreateTemplate(ctx context.Context, template *domain.PromptTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*domain.PromptTemplate, error)
	GetLatestTemplateByName(ctx context.Context, name string) (*domain.PromptTemplate, error)
	ListTemplates(ctx context.Context) ([]*domain.PromptTemplate, error)
	ListTemplateVersions(ctx context.Context, name string) ([]*domain.PromptTemplate, error)
	DeleteTemplate(ctx context.Context, id uuid.UUID) error

	// PromptCollection CRUD
	CreateCollection(ctx context.Context, collection *domain.PromptCollection) error
	GetCollectionByID(ctx context.Context, id uuid.UUID) (*domain.PromptCollection, error)
	ListCollections(ctx context.Context) ([]*domain.PromptCollection, error)
	UpdateCollection(ctx context.Context, collection *domain.PromptCollection) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error

	// Resolution helper
	ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*domain.PromptTemplate, error)
}

// AgentRepository defines the persistent storage operations for Agent templates.
type AgentRepository interface {
	Create(ctx context.Context, agent *domain.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	GetByName(ctx context.Context, name string) (*domain.Agent, error)
	List(ctx context.Context) ([]*domain.Agent, error)
	Update(ctx context.Context, agent *domain.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error

	AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error
	UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error
}

// CommunityRepository defines the persistent storage operations for Community configurations.
type CommunityRepository interface {
	Create(ctx context.Context, community *domain.Community) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Community, error)
	GetByName(ctx context.Context, name string) (*domain.Community, error)
	List(ctx context.Context) ([]*domain.Community, error)
	Update(ctx context.Context, community *domain.Community) error
	Delete(ctx context.Context, id uuid.UUID) error
}


