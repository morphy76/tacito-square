package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCommunityAssignment_Validate(t *testing.T) {
	now := time.Now()
	validAssignment := CommunityAssignment{
		CommunityID: uuid.New(),
		AgentID:     uuid.New(),
		TenantID:    "test-tenant.com",
		Role:        AgentRoleHub,
		AssignedAt:  now,
	}

	tests := []struct {
		name       string
		assignment CommunityAssignment
		wantErr    string
	}{
		{
			name:       "Valid hub assignment",
			assignment: validAssignment,
		},
		{
			name: "Valid spoke assignment",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.Role = AgentRoleSpoke
				return a
			}(),
		},
		{
			name: "Valid standalone assignment",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.Role = AgentRoleStandalone
				return a
			}(),
		},
		{
			name: "Valid assignment with informed_at set",
			assignment: func() CommunityAssignment {
				a := validAssignment
				t := time.Now()
				a.InformedAt = &t
				return a
			}(),
		},
		{
			name: "Missing CommunityID",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.CommunityID = uuid.Nil
				return a
			}(),
			wantErr: "community_id is required",
		},
		{
			name: "Missing AgentID",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.AgentID = uuid.Nil
				return a
			}(),
			wantErr: "agent_id is required",
		},
		{
			name: "Missing TenantID",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.TenantID = ""
				return a
			}(),
			wantErr: "tenant_id is required",
		},
		{
			name: "Invalid role",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.Role = AgentRole("invalid-role")
				return a
			}(),
			wantErr: "invalid agent role",
		},
		{
			name: "Empty role",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.Role = AgentRole("")
				return a
			}(),
			wantErr: "invalid agent role",
		},
		{
			name: "Zero AssignedAt",
			assignment: func() CommunityAssignment {
				a := validAssignment
				a.AssignedAt = time.Time{}
				return a
			}(),
			wantErr: "assigned_at is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.assignment.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAgentRole_Constants(t *testing.T) {
	assert.Equal(t, AgentRole("standalone"), AgentRoleStandalone)
	assert.Equal(t, AgentRole("hub"), AgentRoleHub)
	assert.Equal(t, AgentRole("spoke"), AgentRoleSpoke)
}
