package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

// LLMBindingRepository defines the persistent storage operations for LLM Provider Bindings.
type LLMBindingRepository interface {
	Create(ctx context.Context, binding *model.LLMBinding) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.LLMBinding, error)
	GetByName(ctx context.Context, name string) (*model.LLMBinding, error)
	List(ctx context.Context) ([]*model.LLMBinding, error)
	Update(ctx context.Context, binding *model.LLMBinding) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// MCPClientRepository defines the persistent storage operations for MCP Client configurations.
type MCPClientRepository interface {
	Create(ctx context.Context, client *model.MCPClient) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MCPClient, error)
	GetByName(ctx context.Context, name string) (*model.MCPClient, error)
	List(ctx context.Context) ([]*model.MCPClient, error)
	Update(ctx context.Context, client *model.MCPClient) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SkillRepository defines the persistent storage operations for Skills and Skill Collections.
type SkillRepository interface {
	Create(ctx context.Context, skill *model.Skill) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error)
	GetByName(ctx context.Context, name string) (*model.Skill, error)
	List(ctx context.Context) ([]*model.Skill, error)
	Update(ctx context.Context, skill *model.Skill) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Agent-Skill associations
	AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
	DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error
	ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Skill, error)

	// SkillCollection CRUD
	CreateCollection(ctx context.Context, collection *model.SkillCollection) error
	GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.SkillCollection, error)
	ListCollections(ctx context.Context) ([]*model.SkillCollection, error)
	UpdateCollection(ctx context.Context, collection *model.SkillCollection) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error

	// Resolution helper
	ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error)
}

// PromptRepository defines the persistent storage operations for Prompt Templates and Prompt Collections.
type PromptRepository interface {
	// PromptTemplate CRUD
	CreateTemplate(ctx context.Context, template *model.PromptTemplate) error
	GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error)
	ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error)
	UpdateTemplate(ctx context.Context, template *model.PromptTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error

	// PromptCollection CRUD
	CreateCollection(ctx context.Context, collection *model.PromptCollection) error
	GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error)
	ListCollections(ctx context.Context) ([]*model.PromptCollection, error)
	UpdateCollection(ctx context.Context, collection *model.PromptCollection) error
	DeleteCollection(ctx context.Context, id uuid.UUID) error

	// Resolution helper
	ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error)
}

// AgentRepository defines the persistent storage operations for Agent templates.
type AgentRepository interface {
	Create(ctx context.Context, agent *model.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error)
	GetByName(ctx context.Context, name string) (*model.Agent, error)
	List(ctx context.Context) ([]*model.Agent, error)
	Update(ctx context.Context, agent *model.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error

	AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error
	UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error
}

// CommunityRepository defines the persistent storage operations for Community configurations.
type CommunityRepository interface {
	Create(ctx context.Context, community *model.Community) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error)
	GetByName(ctx context.Context, name string) (*model.Community, error)
	List(ctx context.Context) ([]*model.Community, error)
	Update(ctx context.Context, community *model.Community) error
	Delete(ctx context.Context, id uuid.UUID) error
}
