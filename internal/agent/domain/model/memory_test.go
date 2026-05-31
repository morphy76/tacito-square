package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryEntryValidation(t *testing.T) {
	t.Run("valid memory entry", func(t *testing.T) {
		entry := MemoryEntry{
			Role:      "user",
			Content:   "hello",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing role", func(t *testing.T) {
		entry := MemoryEntry{
			Content:   "hello",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role must not be empty")
	})

	t.Run("missing content", func(t *testing.T) {
		entry := MemoryEntry{
			Role:      "user",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content must not be empty")
	})

	t.Run("invalid role type", func(t *testing.T) {
		entry := MemoryEntry{
			Role:      "invalid_role",
			Content:   "hello",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role type")
	})
}
