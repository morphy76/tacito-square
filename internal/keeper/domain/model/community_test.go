package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCommunity_Validation(t *testing.T) {
	validCommunity := Community{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "qa-community",
		Description: "QA Testing Community",
		Topology:    CommunityTopologyHubSpoke,
		Configuration: map[string]interface{}{
			"max_messages_per_sec": 100,
		},
		Status:    CommunityStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name      string
		community Community
		wantErr   string
	}{
		{
			name:      "Valid community",
			community: validCommunity,
		},
		{
			name: "Missing ID",
			community: func() Community {
				c := validCommunity
				c.ID = uuid.Nil
				return c
			}(),
			wantErr: "id is required",
		},
		{
			name: "Missing Tenant ID",
			community: func() Community {
				c := validCommunity
				c.TenantID = ""
				return c
			}(),
			wantErr: "tenant id is required",
		},
		{
			name: "Missing Name",
			community: func() Community {
				c := validCommunity
				c.Name = ""
				return c
			}(),
			wantErr: "name is required",
		},
		{
			name: "Invalid or unsupported topology",
			community: func() Community {
				c := validCommunity
				c.Topology = CommunityTopology("unsupported")
				return c
			}(),
			wantErr: "invalid or unsupported topology",
		},
		{
			name: "Invalid status",
			community: func() Community {
				c := validCommunity
				c.Status = CommunityStatus("invalid-status")
				return c
			}(),
			wantErr: "invalid community status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.community.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
