package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAgent_Validation(t *testing.T) {
	temp := 0.7
	maxTokens := 2048
	validAgent := Agent{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "qa-agent",
		Description: "QA Testing Agent template",
		Brain: BrainConfig{
			LLMBindingID: uuid.New(),
			Temperature:  &temp,
			MaxTokens:    &maxTokens,
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
		Prompts: []uuid.UUID{uuid.New()},
		PromptCollections: []uuid.UUID{uuid.New()},
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
			name: "Missing LLM Binding ID",
			agent: func() Agent {
				a := validAgent
				a.Brain.LLMBindingID = uuid.Nil
				return a
			}(),
			wantErr: "brain llm binding id is required",
		},
		{
			name: "Invalid brain temperature too low",
			agent: func() Agent {
				a := validAgent
				val := -0.1
				a.Brain.Temperature = &val
				return a
			}(),
			wantErr: "brain temperature must be between 0.0 and 2.0",
		},
		{
			name: "Invalid brain temperature too high",
			agent: func() Agent {
				a := validAgent
				val := 2.1
				a.Brain.Temperature = &val
				return a
			}(),
			wantErr: "brain temperature must be between 0.0 and 2.0",
		},
		{
			name: "Invalid brain max tokens",
			agent: func() Agent {
				a := validAgent
				val := -1
				a.Brain.MaxTokens = &val
				return a
			}(),
			wantErr: "brain max tokens must be non-negative",
		},
		{
			name: "Valid Agent with nil Brain Temperature and MaxTokens overrides",
			agent: func() Agent {
				a := validAgent
				a.Brain.Temperature = nil
				a.Brain.MaxTokens = nil
				return a
			}(),
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
		{
			name: "Valid Agent with empty/zero long-term memory",
			agent: func() Agent {
				a := validAgent
				a.LongTermMemory = LongTermMemoryConfig{
					CollectionName:  "",
					VectorDimension: 0,
				}
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

func TestAgent_JSONSerialization(t *testing.T) {
	a := Agent{
		ID:        uuid.New(),
		Name:      "test-agent",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	data, err := json.Marshal(a)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	assert.NoError(t, err)

	// Omitted empty/nil fields
	assert.NotContains(t, m, "skills")
	assert.NotContains(t, m, "mcp_clients")
	assert.NotContains(t, m, "community_id")
	assert.NotContains(t, m, "description")
	assert.NotContains(t, m, "tier")

	// Present required/non-empty fields
	assert.Contains(t, m, "id")
	assert.Contains(t, m, "name")
}

