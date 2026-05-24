package inbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

// LLMBindingUseCase defines the driving operations for managing LLM Provider Bindings.
type LLMBindingUseCase interface {
	Create(ctx context.Context, binding *model.LLMBinding) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error)
	List(ctx context.Context) ([]*model.LLMBinding, error)
	Update(ctx context.Context, binding *model.LLMBinding) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MCPServerUseCase defines the driving operations for managing MCP Server configurations.
type MCPServerUseCase interface {
	Create(ctx context.Context, server *model.MCPServer) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MCPServer, error)
	List(ctx context.Context) ([]*model.MCPServer, error)
	Update(ctx context.Context, server *model.MCPServer) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SkillUseCase defines the driving operations for managing Skill Collections and associations.
type SkillUseCase interface {
	Create(ctx context.Context, skill *model.Skill) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error)
	List(ctx context.Context) ([]*model.Skill, error)
	Update(ctx context.Context, skill *model.Skill) error
	Delete(ctx context.Context, id uuid.UUID) error

	AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
	DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
}

// PromptUseCase defines the driving operations for managing Prompts and resolution.
type PromptUseCase interface {
	CreateTemplate(ctx context.Context, template *model.PromptTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error)
	GetLatestTemplateByName(ctx context.Context, name string) (*model.PromptTemplate, error)
	ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error)
	DeleteTemplate(ctx context.Context, id uuid.UUID) error

	CreateCollection(ctx context.Context, collection *model.PromptCollection) error
	GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error)
	ListCollections(ctx context.Context) ([]*model.PromptCollection, error)
	UpdateCollection(ctx context.Context, collection *model.PromptCollection) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error
	ResolveCollection(ctx context.Context, id uuid.UUID) ([]*model.PromptTemplate, error)
	ResolveCollectionPrompts(ctx context.Context, id uuid.UUID) ([]*model.PromptTemplate, error)
}

// AgentUseCase defines the driving operations for Agent configurations.
type AgentUseCase interface {
	Create(ctx context.Context, agent *model.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error)
	List(ctx context.Context) ([]*model.Agent, error)
	Update(ctx context.Context, agent *model.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CommunityUseCase defines the driving operations for Community configurations.
type CommunityUseCase interface {
	Create(ctx context.Context, community *model.Community) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error)
	List(ctx context.Context) ([]*model.Community, error)
	Update(ctx context.Context, community *model.Community) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AssignmentUseCase defines the driving operations for Agent-Community assignments.
type AssignmentUseCase interface {
	Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error
	Unassign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error
}
