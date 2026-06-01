package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMCPClient_Validation(t *testing.T) {
	validStdio := MCPClient{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "sqlite-mcp",
		Description: "SQLite MCP client",
		Transport:   TransportStdio,
		Command:     "/usr/local/bin/mcp-sqlite",
		Args:        []string{"--db", "test.db"},
		Env:         map[string]string{"DEBUG": "true"},
		Status:      MCPClientStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	validSSE := MCPClient{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "github-mcp",
		Description: "GitHub MCP client",
		Transport:   TransportSSE,
		URL:         "https://mcp.github.com/events",
		Status:      MCPClientStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name    string
		client  MCPClient
		wantErr string
	}{
		{
			name:   "Valid stdio client",
			client: validStdio,
		},
		{
			name:   "Valid sse client",
			client: validSSE,
		},
		{
			name: "Missing Tenant ID",
			client: func() MCPClient {
				s := validStdio
				s.TenantID = ""
				return s
			}(),
			wantErr: "tenant id is required",
		},
		{
			name: "Missing ID",
			client: func() MCPClient {
				s := validStdio
				s.ID = uuid.Nil
				return s
			}(),
			wantErr: "id is required",
		},
		{
			name: "Missing name",
			client: func() MCPClient {
				s := validStdio
				s.Name = ""
				return s
			}(),
			wantErr: "name is required",
		},
		{
			name: "Invalid transport",
			client: func() MCPClient {
				s := validStdio
				s.Transport = Transport("invalid")
				return s
			}(),
			wantErr: "invalid transport",
		},
		{
			name: "Invalid status",
			client: func() MCPClient {
				s := validStdio
				s.Status = MCPClientStatus("invalid")
				return s
			}(),
			wantErr: "invalid status",
		},
		{
			name: "Stdio transport missing command",
			client: func() MCPClient {
				s := validStdio
				s.Command = ""
				return s
			}(),
			wantErr: "command is required for stdio transport",
		},
		{
			name: "Stdio transport having URL",
			client: func() MCPClient {
				s := validStdio
				s.URL = "http://localhost:8080"
				return s
			}(),
			wantErr: "url must be empty for stdio transport",
		},
		{
			name: "SSE transport missing URL",
			client: func() MCPClient {
				s := validSSE
				s.URL = ""
				return s
			}(),
			wantErr: "url is required for sse transport",
		},
		{
			name: "SSE transport invalid URL format",
			client: func() MCPClient {
				s := validSSE
				s.URL = "invalid-url"
				return s
			}(),
			wantErr: "url must be a valid http or https URL",
		},
		{
			name: "SSE transport having command",
			client: func() MCPClient {
				s := validSSE
				s.Command = "node"
				return s
			}(),
			wantErr: "command must be empty for sse transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
