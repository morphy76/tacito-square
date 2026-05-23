package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PromptRole defines the role field in prompt templates.
type PromptRole string

const (
	PromptRoleSystem    PromptRole = "system"
	PromptRoleUser      PromptRole = "user"
	PromptRoleAssistant PromptRole = "assistant"
)

// PromptStatus defines the lifecycle status of a prompt template version.
type PromptStatus string

const (
	PromptStatusDraft    PromptStatus = "draft"
	PromptStatusActive   PromptStatus = "active"
	PromptStatusArchived PromptStatus = "archived"
)

// PromptTemplate represents a single parameterized instruction block with a role.
type PromptTemplate struct {
	ID        uuid.UUID    `json:"id"`
	Name      string       `json:"name"`
	Content   string       `json:"content"`
	Role      PromptRole   `json:"role"`
	Version   int          `json:"version"`
	Status    PromptStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
}

// PromptCollection represents a suite of templates used together by an agent profile.
type PromptCollection struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Templates   []uuid.UUID `json:"templates"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Validate checks business invariants for PromptTemplate.
func (t PromptTemplate) Validate() error {
	if t.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if t.Name == "" {
		return errors.New("name is required")
	}
	if t.Role != PromptRoleSystem && t.Role != PromptRoleUser && t.Role != PromptRoleAssistant {
		return errors.New("invalid role")
	}
	if t.Status != PromptStatusDraft && t.Status != PromptStatusActive && t.Status != PromptStatusArchived {
		return errors.New("invalid status")
	}
	if t.Version <= 0 {
		return errors.New("version must be greater than zero")
	}
	return nil
}

// Validate checks business invariants for PromptCollection.
func (c PromptCollection) Validate() error {
	if c.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if c.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
