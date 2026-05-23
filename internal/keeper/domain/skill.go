package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SkillStatus represents the lifecycle state of a skill collection.
type SkillStatus string

const (
	SkillStatusActive    SkillStatus = "active"
	SkillStatusSuspended SkillStatus = "suspended"
	SkillStatusInactive  SkillStatus = "inactive"
)

// Skill defines a Skill Collection aggregate grouping tools and server bindings.
type Skill struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	MCPServers   []uuid.UUID     `json:"mcp_servers"`
	AllowedTools []string        `json:"allowed_tools"`
	DeniedTools  []string        `json:"denied_tools"`
	Status       SkillStatus     `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Validate checks all business rules and invariants of the Skill aggregate.
func (s Skill) Validate() error {
	if s.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if s.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Status != SkillStatusActive && s.Status != SkillStatusSuspended && s.Status != SkillStatusInactive {
		return errors.New("invalid status")
	}
	return nil
}

// IsToolAuthorized returns true if the tool name is authorized by whitelists and blacklists.
func (s Skill) IsToolAuthorized(toolName string) bool {
	// 1. Blacklist takes absolute precedence
	for _, denied := range s.DeniedTools {
		if denied == toolName {
			return false
		}
	}

	// 2. If whitelist is empty, all non-blacklisted tools are authorized
	if len(s.AllowedTools) == 0 {
		return true
	}

	// 3. Otherwise, the tool must exist in the whitelist
	for _, allowed := range s.AllowedTools {
		if allowed == toolName {
			return true
		}
	}

	return false
}
