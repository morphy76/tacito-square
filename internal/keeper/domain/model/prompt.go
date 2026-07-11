package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// PromptStatus defines the lifecycle status of a prompt template version.
type PromptStatus string

const (
	PromptStatusDraft    PromptStatus = "draft"
	PromptStatusActive   PromptStatus = "active"
	PromptStatusArchived PromptStatus = "archived"
)

// HubSystemPromptTemplateID is the system prompt template UUID.
// It starts with ffffffff to mark it as system-locked.
var HubSystemPromptTemplateID = uuid.MustParse("ffffffff-0000-0000-0000-000000000001")

// IsSystemLocked checks if a UUID represents a system-locked template.
// A UUID is system-locked if its first 4 bytes are 0xFF (starts with ffffffff).
func IsSystemLocked(id uuid.UUID) bool {
	return id[0] == 0xff && id[1] == 0xff && id[2] == 0xff && id[3] == 0xff
}

// PromptTemplate represents a single parameterized instruction block.
type PromptTemplate struct {
	ID        uuid.UUID    `json:"id"`
	TenantID  string       `json:"tenant_id"`
	Name      string       `json:"name"`
	Content   string       `json:"content"`
	Status    PromptStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
}

// PromptCollection represents a suite of templates used together by an agent profile.
type PromptCollection struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    string      `json:"tenant_id,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Templates   []uuid.UUID `json:"templates,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Validate checks business invariants for PromptTemplate.
func (t PromptTemplate) Validate() error {
	if t.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if t.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if t.Name == "" {
		return errors.New("name is required")
	}
	if t.Status != PromptStatusDraft && t.Status != PromptStatusActive && t.Status != PromptStatusArchived {
		return errors.New("invalid status")
	}
	return nil
}

// Validate checks business invariants for PromptCollection.
func (c PromptCollection) Validate() error {
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

// PromptVersion represents a specific immutable content snapshot of a prompt template.
type PromptVersion struct {
	ID              uuid.UUID `json:"id"`
	PromptID        uuid.UUID `json:"prompt_id"`
	VersionNumber   int       `json:"version_number"`
	ContentSnapshot string    `json:"content_snapshot"`
	CreatedAt       time.Time `json:"created_at"`
}

// Validate checks business invariants for PromptVersion.
func (v PromptVersion) Validate() error {
	if v.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if v.PromptID == uuid.Nil {
		return errors.New("prompt id is required")
	}
	if v.VersionNumber <= 0 {
		return errors.New("version number must be positive")
	}
	return nil
}
