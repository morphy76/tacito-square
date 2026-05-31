package model

import (
	"errors"
	"time"
)

// MemoryEntry represents a single conversational or execution step in a thread.
type MemoryEntry struct {
	Role      string            `json:"role"`                 // e.g., "system", "user", "assistant", "tool"
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`   // Type descriptors, tool IDs, execution steps
}

// Validate ensures standard memory entry constraints are met.
func (e *MemoryEntry) Validate() error {
	if e.Role == "" {
		return errors.New("role must not be empty")
	}
	if e.Content == "" {
		return errors.New("content must not be empty")
	}
	switch e.Role {
	case "system", "user", "assistant", "tool":
		// valid
	default:
		return errors.New("invalid role type: " + e.Role)
	}
	return nil
}
