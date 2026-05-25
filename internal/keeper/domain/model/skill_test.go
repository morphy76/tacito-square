package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSkill_Validation(t *testing.T) {
	validSkill := Skill{
		ID:          uuid.New(),
		TenantID:    "test-tenant.com",
		Name:        "web-search",
		Description: "Provides google search tools",
		Status:      SkillStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tests := []struct {
		name    string
		skill   Skill
		wantErr string
	}{
		{
			name:  "Valid skill",
			skill: validSkill,
		},
		{
			name: "Missing Tenant ID",
			skill: func() Skill {
				s := validSkill
				s.TenantID = ""
				return s
			}(),
			wantErr: "tenant id is required",
		},
		{
			name: "Missing ID",
			skill: func() Skill {
				s := validSkill
				s.ID = uuid.Nil
				return s
			}(),
			wantErr: "id is required",
		},
		{
			name: "Missing name",
			skill: func() Skill {
				s := validSkill
				s.Name = ""
				return s
			}(),
			wantErr: "name is required",
		},
		{
			name: "Invalid status",
			skill: func() Skill {
				s := validSkill
				s.Status = SkillStatus("invalid")
				return s
			}(),
			wantErr: "invalid status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSkillCollection_Validate(t *testing.T) {
	t.Run("Valid Skill Collection", func(t *testing.T) {
		pc := SkillCollection{
			ID:          uuid.New(),
			TenantID:    "test-tenant.com",
			Name:        "agent-a-skills",
			Description: "Collection for agent A",
			Skills:      []uuid.UUID{uuid.New()},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		assert.NoError(t, pc.Validate())
	})

	t.Run("Missing Tenant ID", func(t *testing.T) {
		pc := SkillCollection{
			ID:       uuid.New(),
			TenantID: "",
			Name:     "agent-a-skills",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "tenant id is required")
	})

	t.Run("Missing ID", func(t *testing.T) {
		pc := SkillCollection{
			ID:       uuid.Nil,
			TenantID: "test-tenant.com",
			Name:     "agent-a-skills",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		pc := SkillCollection{
			ID:       uuid.New(),
			TenantID: "test-tenant.com",
			Name:     "",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "name is required")
	})
}
