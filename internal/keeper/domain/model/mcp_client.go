package model

import (
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// Transport represents the Model Context Protocol communication transport type.
type Transport string

const (
	TransportStdio Transport = "stdio"
	TransportSSE   Transport = "sse"
)

// MCPClientStatus represents the lifecycle state of the MCP client configuration.
type MCPClientStatus string

const (
	MCPClientStatusActive    MCPClientStatus = "active"
	MCPClientStatusSuspended MCPClientStatus = "suspended"
	MCPClientStatusInactive  MCPClientStatus = "inactive"
)

// MCPClient defines the configuration needed for an agent to communicate with an MCP client/server gateway.
type MCPClient struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      string          `json:"tenant_id,omitempty"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Transport     Transport       `json:"transport"`
	Command       string          `json:"command,omitempty"`
	Args          []string        `json:"args,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	URL           string          `json:"url,omitempty"`
	AuthSecretRef string          `json:"auth_secret_ref,omitempty"`
	Status        MCPClientStatus `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// Validate checks all business rules and invariants of the MCPClient aggregate.
func (c MCPClient) Validate() error {
	if c.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if c.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	if c.Transport != TransportStdio && c.Transport != TransportSSE {
		return errors.New("invalid transport")
	}
	if c.Status != MCPClientStatusActive && c.Status != MCPClientStatusSuspended && c.Status != MCPClientStatusInactive {
		return errors.New("invalid status")
	}

	if c.Transport == TransportStdio {
		if c.Command == "" {
			return errors.New("command is required for stdio transport")
		}
		if c.URL != "" {
			return errors.New("url must be empty for stdio transport")
		}
	}

	if c.Transport == TransportSSE {
		if c.URL == "" {
			return errors.New("url is required for sse transport")
		}
		parsed, err := url.Parse(c.URL)
		if err != nil || parsed.Scheme == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("url must be a valid http or https URL")
		}
		if c.Command != "" {
			return errors.New("command must be empty for sse transport")
		}
	}

	return nil
}
