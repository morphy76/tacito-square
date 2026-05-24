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

// MCPServerStatus represents the lifecycle state of the MCP server configuration.
type MCPServerStatus string

const (
	MCPServerStatusActive    MCPServerStatus = "active"
	MCPServerStatusSuspended MCPServerStatus = "suspended"
	MCPServerStatusInactive  MCPServerStatus = "inactive"
)

// MCPServer defines the configuration needed for an agent to communicate with an MCP Server.
type MCPServer struct {
	ID            uuid.UUID         `json:"id"`
	TenantID      string            `json:"tenant_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Transport     Transport         `json:"transport"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	URL           string            `json:"url"`
	AuthSecretRef string            `json:"auth_secret_ref"`
	Status        MCPServerStatus   `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Validate checks all business rules and invariants of the MCPServer aggregate.
func (s MCPServer) Validate() error {
	if s.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if s.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Transport != TransportStdio && s.Transport != TransportSSE {
		return errors.New("invalid transport")
	}
	if s.Status != MCPServerStatusActive && s.Status != MCPServerStatusSuspended && s.Status != MCPServerStatusInactive {
		return errors.New("invalid status")
	}

	if s.Transport == TransportStdio {
		if s.Command == "" {
			return errors.New("command is required for stdio transport")
		}
		if s.URL != "" {
			return errors.New("url must be empty for stdio transport")
		}
	}

	if s.Transport == TransportSSE {
		if s.URL == "" {
			return errors.New("url is required for sse transport")
		}
		parsed, err := url.Parse(s.URL)
		if err != nil || parsed.Scheme == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("url must be a valid http or https URL")
		}
		if s.Command != "" {
			return errors.New("command must be empty for sse transport")
		}
	}

	return nil
}
