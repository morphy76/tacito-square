package domain

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
)

// BrainConfig encapsulates Large Language Model settings for the agent.
type BrainConfig struct {
	Model             string  `json:"model"`
	Temperature       float64 `json:"temperature"`
	MaxTokens         int     `json:"max_tokens"`
	Endpoint          string  `json:"endpoint"`
	CredentialsSecret string  `json:"credentials_secret"`
}

// ShortTermMemoryConfig encapsulates Redis ephemeral state configuration.
type ShortTermMemoryConfig struct {
	KeyNamespace string `json:"key_namespace"`
	TTLSeconds   int    `json:"ttl_seconds"`
}

// LongTermMemoryConfig encapsulates Qdrant semantic storage configuration.
type LongTermMemoryConfig struct {
	CollectionName  string `json:"collection_name"`
	VectorDimension int    `json:"vector_dimension"`
}

// MCPClientConfig encapsulates custom configurations for attached MCP servers.
type MCPClientConfig struct {
	ServerID   uuid.UUID         `json:"server_id"`
	CustomEnv  map[string]string `json:"custom_env"`
	CustomArgs []string          `json:"custom_args"`
}

// Agent represents the aggregate root for an Agent Template within Keeper.
type Agent struct {
	ID              uuid.UUID             `json:"id"`
	TenantID        string                `json:"tenant_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Brain           BrainConfig           `json:"brain"`
	ShortTermMemory ShortTermMemoryConfig `json:"short_term_memory"`
	LongTermMemory  LongTermMemoryConfig  `json:"long_term_memory"`
	Skills          []uuid.UUID           `json:"skills"`
	PromptTemplate  uuid.UUID             `json:"prompt_template"`
	MCPClients      []MCPClientConfig     `json:"mcp_clients"`
	Status          AgentStatus           `json:"status"`
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
	if a.Status != AgentStatusDefined && a.Status != AgentStatusAssigned && a.Status != AgentStatusActive && a.Status != AgentStatusTerminated {
		return errors.New("invalid agent status")
	}

	// Brain validations
	if a.Brain.Model == "" {
		return errors.New("brain model is required")
	}
	if a.Brain.Temperature < 0.0 || a.Brain.Temperature > 2.0 {
		return errors.New("brain temperature must be between 0.0 and 2.0")
	}
	if a.Brain.MaxTokens <= 0 {
		return errors.New("brain max tokens must be positive")
	}

	// ShortTermMemory validations
	if a.ShortTermMemory.TTLSeconds <= 0 {
		return errors.New("short-term memory ttl must be positive")
	}

	// LongTermMemory validations
	if a.LongTermMemory.VectorDimension <= 0 {
		return errors.New("long-term memory vector dimension must be positive")
	}

	// MCPClients validations
	for _, mcp := range a.MCPClients {
		if mcp.ServerID == uuid.Nil {
			return errors.New("mcp client server id is required")
		}
	}

	return nil
}
