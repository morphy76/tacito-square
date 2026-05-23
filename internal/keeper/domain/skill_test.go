package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSkill_Validation(t *testing.T) {
	validSkill := Skill{
		ID:          uuid.New(),
		Name:        "web-search",
		Description: "Provides google search tools",
		MCPServers:  []uuid.UUID{uuid.New()},
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

func TestSkill_IsToolAuthorized(t *testing.T) {
	tests := []struct {
		name         string
		allowedTools []string
		deniedTools  []string
		testTool     string
		wantAuth     bool
	}{
		{
			name:         "All allowed by default (both empty)",
			allowedTools: nil,
			deniedTools:  nil,
			testTool:     "search_google",
			wantAuth:     true,
		},
		{
			name:         "Allowed in whitelist, not blacklisted",
			allowedTools: []string{"search_google", "fetch_url"},
			deniedTools:  nil,
			testTool:     "search_google",
			wantAuth:     true,
		},
		{
			name:         "Not in whitelist",
			allowedTools: []string{"search_google", "fetch_url"},
			deniedTools:  nil,
			testTool:     "read_file",
			wantAuth:     false,
		},
		{
			name:         "In blacklist, whitelist empty",
			allowedTools: nil,
			deniedTools:  []string{"format_disk", "read_file"},
			testTool:     "format_disk",
			wantAuth:     false,
		},
		{
			name:         "Not in blacklist, whitelist empty",
			allowedTools: nil,
			deniedTools:  []string{"format_disk"},
			testTool:     "read_file",
			wantAuth:     true,
		},
		{
			name:         "In whitelist but also in blacklist",
			allowedTools: []string{"search_google", "fetch_url"},
			deniedTools:  []string{"fetch_url"},
			testTool:     "fetch_url",
			wantAuth:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := Skill{
				AllowedTools: tt.allowedTools,
				DeniedTools:  tt.deniedTools,
			}
			assert.Equal(t, tt.wantAuth, skill.IsToolAuthorized(tt.testTool))
		})
	}
}
