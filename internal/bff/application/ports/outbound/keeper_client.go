package outbound

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// KeeperClient defines the outbound port for interacting with the Keeper core services,
// composed of resource-specific sub-clients for better maintainability and separation of concerns.
type KeeperClient interface {
	Ping(ctx context.Context) error
	LLMBindings() LLMBindingClient
	MCPServers() MCPServerClient
	Skills() SkillClient
	Prompts() PromptClient
	Agents() AgentClient
	Communities() CommunityClient
}

// LLMBindingClient defines the outbound interface for LLM provider binding CRUD operations.
type LLMBindingClient interface {
	Create(ctx context.Context, req *CreateLLMBindingRequest) (*LLMBinding, error)
	Get(ctx context.Context, id uuid.UUID) (*LLMBinding, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateLLMBindingRequest) (*LLMBinding, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*LLMBinding, error)
}

// MCPServerClient defines the outbound interface for MCP server registration CRUD operations.
type MCPServerClient interface {
	Create(ctx context.Context, req *CreateMCPServerRequest) (*MCPServer, error)
	Get(ctx context.Context, id uuid.UUID) (*MCPServer, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateMCPServerRequest) (*MCPServer, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*MCPServer, error)
}

// SkillClient defines the outbound interface for skill definition CRUD operations.
type SkillClient interface {
	Create(ctx context.Context, req *CreateSkillRequest) (*Skill, error)
	Get(ctx context.Context, id uuid.UUID) (*Skill, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateSkillRequest) (*Skill, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Skill, error)
}

// PromptClient defines the outbound interface for prompt template CRUD operations.
type PromptClient interface {
	Create(ctx context.Context, req *CreatePromptTemplateRequest) (*PromptTemplate, error)
	Get(ctx context.Context, id uuid.UUID) (*PromptTemplate, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdatePromptTemplateRequest) (*PromptTemplate, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*PromptTemplate, error)
}

// AgentClient defines the outbound interface for agent template CRUD operations.
type AgentClient interface {
	Create(ctx context.Context, req *CreateAgentRequest) (*Agent, error)
	Get(ctx context.Context, id uuid.UUID) (*Agent, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateAgentRequest) (*Agent, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Agent, error)
}

// CommunityClient defines the outbound interface for community boundary CRUD and assignment operations.
type CommunityClient interface {
	Create(ctx context.Context, req *CreateCommunityRequest) (*Community, error)
	Get(ctx context.Context, id uuid.UUID) (*Community, error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateCommunityRequest) (*Community, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*Community, error)
	AssignAgent(ctx context.Context, communityID, agentID uuid.UUID) error
	UnassignAgent(ctx context.Context, communityID, agentID uuid.UUID) error
}

// --- LLM Binding DTOs ---

type LLMBinding struct {
	ID                 uuid.UUID `json:"id"`
	TenantID           string    `json:"tenant_id,omitempty"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	Provider           string    `json:"provider"`
	APIBaseURL         string    `json:"api_base_url"`
	APIKeySecretRef    string    `json:"api_key_secret_ref"`
	DefaultModel       string    `json:"default_model"`
	DefaultTemperature float64   `json:"default_temperature"`
	DefaultMaxTokens   int       `json:"default_max_tokens"`
	TimeoutSeconds     int       `json:"timeout_seconds"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateLLMBindingRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	Provider           string   `json:"provider"`
	APIBaseURL         string   `json:"api_base_url"`
	APIKeySecretRef    string   `json:"api_key_secret_ref"`
	DefaultModel       string   `json:"default_model"`
	DefaultTemperature *float64 `json:"default_temperature,omitempty"`
	DefaultMaxTokens   int      `json:"default_max_tokens,omitempty"`
	TimeoutSeconds     int      `json:"timeout_seconds,omitempty"`
}

type UpdateLLMBindingRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description,omitempty"`
	Provider           string   `json:"provider"`
	APIBaseURL         string   `json:"api_base_url"`
	APIKeySecretRef    string   `json:"api_key_secret_ref"`
	DefaultModel       string   `json:"default_model"`
	DefaultTemperature *float64 `json:"default_temperature,omitempty"`
	DefaultMaxTokens   int      `json:"default_max_tokens,omitempty"`
	TimeoutSeconds     int      `json:"timeout_seconds,omitempty"`
}

// --- MCP Server DTOs ---

type MCPServer struct {
	ID            uuid.UUID         `json:"id"`
	TenantID      string            `json:"tenant_id,omitempty"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Transport     string            `json:"transport"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	URL           string            `json:"url,omitempty"`
	AuthSecretRef string            `json:"auth_secret_ref,omitempty"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type CreateMCPServerRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Transport     string            `json:"transport"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	URL           string            `json:"url,omitempty"`
	AuthSecretRef string            `json:"auth_secret_ref,omitempty"`
}

type UpdateMCPServerRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Transport     string            `json:"transport"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	URL           string            `json:"url,omitempty"`
	AuthSecretRef string            `json:"auth_secret_ref,omitempty"`
}

// --- Skill DTOs ---

type Skill struct {
	ID          uuid.UUID `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateSkillRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
}

type UpdateSkillRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
}

// --- Prompt Template DTOs ---

type PromptTemplate struct {
	ID        uuid.UUID `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePromptTemplateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
	Status  string `json:"status,omitempty"`
}

type UpdatePromptTemplateRequest struct {
	Name    string `json:"name"`
	Content string `json:"content,omitempty"`
	Status  string `json:"status"`
}

// --- Agent DTOs ---

type BrainConfig struct {
	LLMBindingID uuid.UUID `json:"llm_binding_id"`
	Temperature  *float64  `json:"temperature,omitempty"`
	MaxTokens    *int      `json:"max_tokens,omitempty"`
}

type ShortTermMemoryConfig struct {
	KeyNamespace string `json:"key_namespace"`
	TTLSeconds   int    `json:"ttl_seconds"`
}

type LongTermMemoryConfig struct {
	CollectionName  string `json:"collection_name"`
	VectorDimension int    `json:"vector_dimension"`
}

type MCPClientConfig struct {
	ClientID     uuid.UUID         `json:"client_id"`
	CustomEnv    map[string]string `json:"custom_env,omitempty"`
	CustomArgs   []string          `json:"custom_args,omitempty"`
	AllowedTools []string          `json:"allowed_tools,omitempty"`
}

type Agent struct {
	ID              uuid.UUID             `json:"id"`
	TenantID        string                `json:"tenant_id,omitempty"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Role            string                `json:"role,omitempty"`
	Brain           BrainConfig           `json:"brain"`
	ShortTermMemory ShortTermMemoryConfig `json:"short_term_memory"`
	LongTermMemory  LongTermMemoryConfig  `json:"long_term_memory"`
	Skills          []uuid.UUID           `json:"skills,omitempty"`
	PromptTemplate  uuid.UUID             `json:"prompt_template,omitempty"`
	MCPClients      []MCPClientConfig     `json:"mcp_clients,omitempty"`
	Status          string                `json:"status"`
	CommunityID     *uuid.UUID            `json:"community_id,omitempty"`
	Tier            string                `json:"tier,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type CreateBrainRequest struct {
	LLMBindingID string   `json:"llm_binding_id"`
	Temperature  *float64 `json:"temperature,omitempty"`
	MaxTokens    *int     `json:"max_tokens,omitempty"`
}

type CreateShortTermRequest struct {
	KeyNamespace string `json:"key_namespace"`
	TTLSeconds   int    `json:"ttl_seconds"`
}

type CreateLongTermRequest struct {
	CollectionName  string `json:"collection_name"`
	VectorDimension int    `json:"vector_dimension"`
}

type CreateMCPClient struct {
	ClientID     string            `json:"client_id"`
	CustomEnv    map[string]string `json:"custom_env,omitempty"`
	CustomArgs   []string          `json:"custom_args,omitempty"`
	AllowedTools []string          `json:"allowed_tools,omitempty"`
}

type DeploymentRequest struct {
	Tier string `json:"tier,omitempty"`
}

type CreateAgentRequest struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Role            string                 `json:"role,omitempty"`
	Brain           CreateBrainRequest     `json:"brain"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory"`
	Skills          []string               `json:"skills,omitempty"`
	PromptTemplate  string                 `json:"prompt_template,omitempty"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients,omitempty"`
	Deployment      DeploymentRequest      `json:"deployment,omitempty"`
}

type UpdateAgentRequest struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Role            string                 `json:"role,omitempty"`
	Brain           CreateBrainRequest     `json:"brain"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory"`
	Skills          []string               `json:"skills,omitempty"`
	PromptTemplate  string                 `json:"prompt_template,omitempty"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients,omitempty"`
	Deployment      DeploymentRequest      `json:"deployment,omitempty"`
}

// --- Community DTOs ---

type Community struct {
	ID            uuid.UUID              `json:"id"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Topology      string                 `json:"topology"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type CreateCommunityRequest struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Topology      string                 `json:"topology"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

type UpdateCommunityRequest struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Topology      string                 `json:"topology"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Status        string                 `json:"status,omitempty"`
}
