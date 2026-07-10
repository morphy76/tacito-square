package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// AgentStatus represents the lifecycle state of an Agent template or container.
type AgentStatus string

const (
	AgentStatusDefined    AgentStatus = "defined"
	AgentStatusAssigned   AgentStatus = "assigned"
	AgentStatusActive     AgentStatus = "active"
	AgentStatusTerminated AgentStatus = "terminated"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusPending    AgentStatus = "pending"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusError      AgentStatus = "error"
)

// BrainConfig encapsulates Large Language Model settings for the agent.
type BrainConfig struct {
	LLMBindingID uuid.UUID `json:"llm_binding_id"`
	Temperature  *float64  `json:"temperature,omitempty"`
	MaxTokens    *int      `json:"max_tokens,omitempty"`
}

// ShortTermMemoryConfig encapsulates Redis ephemeral state configuration.
type ShortTermMemoryConfig struct {
	KeyNamespace string `json:"key_namespace"`
	TTLSeconds   int    `json:"ttl_seconds"`
}

// LongTermMemoryConfig encapsulates Qdrant semantic storage configuration.
type LongTermMemoryConfig struct {
	CollectionName  string `json:"collection_name,omitempty"`
	VectorDimension int    `json:"vector_dimension,omitempty"`
}

// MCPClientConfig encapsulates custom configurations for attached MCP clients.
type MCPClientConfig struct {
	ClientID     uuid.UUID         `json:"client_id"`
	CustomEnv    map[string]string `json:"custom_env,omitempty"`
	CustomArgs   []string          `json:"custom_args,omitempty"`
	AllowedTools []string          `json:"allowed_tools,omitempty"`
}

// Agent represents the aggregate root for an Agent Template within Keeper.
type Agent struct {
	ID              uuid.UUID             `json:"id"`
	TenantID        string                `json:"tenant_id,omitempty"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	Brain           BrainConfig           `json:"brain"`
	ShortTermMemory ShortTermMemoryConfig `json:"short_term_memory"`
	LongTermMemory  LongTermMemoryConfig  `json:"long_term_memory,omitempty"`
	Skills          []uuid.UUID           `json:"skills,omitempty"`
	PromptTemplate  uuid.UUID             `json:"prompt_template,omitempty"`
	MCPClients      []MCPClientConfig     `json:"mcp_clients,omitempty"`
	Status          AgentStatus           `json:"status"`
	CommunityID     *uuid.UUID            `json:"community_id,omitempty"`
	Tier            string                `json:"tier,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// Validate checks business invariants of the Agent template aggregate.
func (a Agent) Validate() error {
	if a.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if a.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if a.Name == "" {
		return errors.New("name is required")
	}

	if a.Status != AgentStatusDefined && a.Status != AgentStatusAssigned && a.Status != AgentStatusActive && a.Status != AgentStatusTerminated &&
		a.Status != AgentStatusStopped && a.Status != AgentStatusPending && a.Status != AgentStatusRunning && a.Status != AgentStatusError {
		return errors.New("invalid agent status")
	}
	if a.CommunityID != nil && a.Status == AgentStatusDefined {
		return errors.New("assigned agent must not have defined status")
	}
	if a.CommunityID == nil && (a.Status == AgentStatusAssigned || a.Status == AgentStatusStopped || a.Status == AgentStatusPending || a.Status == AgentStatusRunning || a.Status == AgentStatusError) {
		return errors.New("unassigned agent must not have assigned status")
	}

	// Brain validations
	if a.Brain.LLMBindingID == uuid.Nil {
		return errors.New("brain llm binding id is required")
	}
	if a.Brain.Temperature != nil && (*a.Brain.Temperature < 0.0 || *a.Brain.Temperature > 2.0) {
		return errors.New("brain temperature must be between 0.0 and 2.0")
	}
	if a.Brain.MaxTokens != nil && *a.Brain.MaxTokens < 0 {
		return errors.New("brain max tokens must be non-negative")
	}

	// ShortTermMemory validations
	if a.ShortTermMemory.TTLSeconds <= 0 {
		return errors.New("short-term memory ttl must be positive")
	}

	// LongTermMemory validations
	if a.LongTermMemory.VectorDimension < 0 {
		return errors.New("long-term memory vector dimension must be non-negative")
	}

	// MCPClients validations
	for _, mcp := range a.MCPClients {
		if mcp.ClientID == uuid.Nil {
			return errors.New("mcp client id is required")
		}
	}

	return nil
}
