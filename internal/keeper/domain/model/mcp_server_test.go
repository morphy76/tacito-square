package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMCPServer_Validation(t *testing.T) {
	validStdio := MCPServer{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "sqlite-mcp",
		Description: "SQLite MCP server",
		Transport:   TransportStdio,
		Command:     "/usr/local/bin/mcp-sqlite",
		Args:        []string{"--db", "test.db"},
		Env:         map[string]string{"DEBUG": "true"},
		Status:      MCPServerStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	validSSE := MCPServer{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "github-mcp",
		Description: "GitHub MCP server",
		Transport:   TransportSSE,
		URL:         "https://mcp.github.com/events",
		Status:      MCPServerStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name    string
		server  MCPServer
		wantErr string
	}{
		{
			name:   "Valid stdio server",
			server: validStdio,
		},
		{
			name:   "Valid sse server",
			server: validSSE,
		},
		{
			name: "Missing Tenant ID",
			server: func() MCPServer {
				s := validStdio
				s.TenantID = ""
				return s
			}(),
			wantErr: "tenant id is required",
		},
		{
			name: "Missing ID",
			server: func() MCPServer {
				s := validStdio
				s.ID = uuid.Nil
				return s
			}(),
			wantErr: "id is required",
		},
		{
			name: "Missing name",
			server: func() MCPServer {
				s := validStdio
				s.Name = ""
				return s
			}(),
			wantErr: "name is required",
		},
		{
			name: "Invalid transport",
			server: func() MCPServer {
				s := validStdio
				s.Transport = Transport("invalid")
				return s
			}(),
			wantErr: "invalid transport",
		},
		{
			name: "Invalid status",
			server: func() MCPServer {
				s := validStdio
				s.Status = MCPServerStatus("invalid")
				return s
			}(),
			wantErr: "invalid status",
		},
		{
			name: "Stdio transport missing command",
			server: func() MCPServer {
				s := validStdio
				s.Command = ""
				return s
			}(),
			wantErr: "command is required for stdio transport",
		},
		{
			name: "Stdio transport having URL",
			server: func() MCPServer {
				s := validStdio
				s.URL = "http://localhost:8080"
				return s
			}(),
			wantErr: "url must be empty for stdio transport",
		},
		{
			name: "SSE transport missing URL",
			server: func() MCPServer {
				s := validSSE
				s.URL = ""
				return s
			}(),
			wantErr: "url is required for sse transport",
		},
		{
			name: "SSE transport invalid URL format",
			server: func() MCPServer {
				s := validSSE
				s.URL = "invalid-url"
				return s
			}(),
			wantErr: "url must be a valid http or https URL",
		},
		{
			name: "SSE transport having command",
			server: func() MCPServer {
				s := validSSE
				s.Command = "node"
				return s
			}(),
			wantErr: "command must be empty for sse transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
