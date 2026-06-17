package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// SkillStatus represents the lifecycle state of a skill.
type SkillStatus string

const (
	SkillStatusActive    SkillStatus = "active"
	SkillStatusSuspended SkillStatus = "suspended"
	SkillStatusInactive  SkillStatus = "inactive"
)

// Skill defines a single Skill capability.
type Skill struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    string      `json:"tenant_id,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Content     string      `json:"content,omitempty"`
	Status      SkillStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// SkillCollection represents a group of skill capabilities.
type SkillCollection struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    string      `json:"tenant_id,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Skills      []uuid.UUID `json:"skills,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
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

// Validate checks all business rules and invariants of the SkillCollection aggregate.
func (c SkillCollection) Validate() error {
	if c.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if c.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
