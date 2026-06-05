package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
)

// MCPClientInfo defines the configuration details for a single target MCP connection.
type MCPClientInfo struct {
	Name         string            `json:"name"`
	Transport    string            `json:"transport"` // "stdio" or "sse"
	Command      string            `json:"command,omitempty"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	URL          string            `json:"url,omitempty"`
	AllowedTools []string          `json:"allowed_tools"`
}

// ToolExecutor abstracts the execution and management of MCP tools.
type ToolExecutor interface {
	// ListAllowedTools returns the consolidated list of all whitelisted tools across all configured MCP clients.
	ListAllowedTools(ctx context.Context) ([]model.ToolDefinition, error)

	// Execute routes and runs a whitelisted tool with the given arguments.
	Execute(ctx context.Context, toolName string, arguments map[string]any) (string, error)

	// Close cleans up all active subprocesses and SSE HTTP connections.
	Close(ctx context.Context) error
}

