package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAgent_Validation(t *testing.T) {
	validAgent := Agent{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "qa-agent",
		Description: "QA Testing Agent template",
		Brain: BrainConfig{
			Model:              "gpt-4o",
			Temperature:        0.7,
			MaxTokens:          2048,
			Endpoint:           "https://api.openai.com/v1",
			CredentialsSecret:  "openai-secret-key",
		},
		ShortTermMemory: ShortTermMemoryConfig{
			KeyNamespace: "agent:qa:short",
			TTLSeconds:   3600,
		},
		LongTermMemory: LongTermMemoryConfig{
			CollectionName:  "agent-qa-long",
			VectorDimension: 1536,
		},
		Skills: []uuid.UUID{uuid.New()},
		PromptTemplate: uuid.New(),
		MCPClients: []MCPClientConfig{
			{
				ClientID: uuid.New(),
			},
		},
		Status:    AgentStatusDefined,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name    string
		agent   Agent
		wantErr string
	}{
		{
			name:  "Valid agent template",
			agent: validAgent,
		},
		{
			name: "Valid Agent with hub role",
			agent: func() Agent {
				a := validAgent
				a.Role = "hub"
				return a
			}(),
		},
		{
			name: "Valid Agent with spoke role",
			agent: func() Agent {
				a := validAgent
				a.Role = "spoke"
				return a
			}(),
		},
		{
			name: "Invalid agent role",
			agent: func() Agent {
				a := validAgent
				a.Role = "invalid-role"
				return a
			}(),
			wantErr: "invalid agent role",
		},
		{
			name: "Missing ID",
			agent: func() Agent {
				a := validAgent
				a.ID = uuid.Nil
				return a
			}(),
			wantErr: "id is required",
		},
		{
			name: "Missing Tenant ID",
			agent: func() Agent {
				a := validAgent
				a.TenantID = ""
				return a
			}(),
			wantErr: "tenant id is required",
		},
		{
			name: "Missing name",
			agent: func() Agent {
				a := validAgent
				a.Name = ""
				return a
			}(),
			wantErr: "name is required",
		},
		{
			name: "Invalid brain model",
			agent: func() Agent {
				a := validAgent
				a.Brain.Model = ""
				return a
			}(),
			wantErr: "brain model is required",
		},
		{
			name: "Missing brain endpoint",
			agent: func() Agent {
				a := validAgent
				a.Brain.Endpoint = ""
				return a
			}(),
			wantErr: "brain endpoint is required",
		},
		{
			name: "Invalid brain endpoint URL",
			agent: func() Agent {
				a := validAgent
				a.Brain.Endpoint = "not-a-valid-url"
				return a
			}(),
			wantErr: "brain endpoint must be a valid URL",
		},
		{
			name: "Missing brain credentials secret",
			agent: func() Agent {
				a := validAgent
				a.Brain.CredentialsSecret = ""
				return a
			}(),
			wantErr: "brain credentials secret is required",
		},
		{
			name: "Invalid brain temperature too low",
			agent: func() Agent {
				a := validAgent
				a.Brain.Temperature = -0.1
				return a
			}(),
			wantErr: "brain temperature must be between 0.0 and 2.0",
		},
		{
			name: "Invalid brain temperature too high",
			agent: func() Agent {
				a := validAgent
				a.Brain.Temperature = 2.1
				return a
			}(),
			wantErr: "brain temperature must be between 0.0 and 2.0",
		},
		{
			name: "Invalid brain max tokens",
			agent: func() Agent {
				a := validAgent
				a.Brain.MaxTokens = 0
				return a
			}(),
			wantErr: "brain max tokens must be positive",
		},
		{
			name: "Invalid short term memory ttl",
			agent: func() Agent {
				a := validAgent
				a.ShortTermMemory.TTLSeconds = 0
				return a
			}(),
			wantErr: "short-term memory ttl must be positive",
		},
		{
			name: "Invalid long term memory vector dimension",
			agent: func() Agent {
				a := validAgent
				a.LongTermMemory.VectorDimension = -1
				return a
			}(),
			wantErr: "long-term memory vector dimension must be positive",
		},
		{
			name: "Invalid status",
			agent: func() Agent {
				a := validAgent
				a.Status = AgentStatus("invalid")
				return a
			}(),
			wantErr: "invalid agent status",
		},
		{
			name: "Invalid MCP client server ID",
			agent: func() Agent {
				a := validAgent
				a.MCPClients = []MCPClientConfig{{ClientID: uuid.Nil}}
				return a
			}(),
			wantErr: "mcp client id is required",
		},
		{
			name: "Valid Agent with Community ID",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatusAssigned
				return a
			}(),
		},
		{
			name: "Assigned status with missing Community ID",
			agent: func() Agent {
				a := validAgent
				a.Status = AgentStatusAssigned
				a.CommunityID = nil
				return a
			}(),
			wantErr: "unassigned agent must not have assigned status",
		},
		{
			name: "Defined status with Community ID",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatusDefined
				return a
			}(),
			wantErr: "assigned agent must not have defined status",
		},
		{
			name: "Valid Agent with stopped status assigned to community",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatus("stopped")
				return a
			}(),
		},
		{
			name: "Valid Agent with pending status assigned to community",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatus("pending")
				return a
			}(),
		},
		{
			name: "Valid Agent with running status assigned to community",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatus("running")
				return a
			}(),
		},
		{
			name: "Valid Agent with error status assigned to community",
			agent: func() Agent {
				a := validAgent
				cid := uuid.New()
				a.CommunityID = &cid
				a.Status = AgentStatus("error")
				return a
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
