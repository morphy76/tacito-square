package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPromptTemplate_Validate(t *testing.T) {
	t.Run("Valid Prompt Template", func(t *testing.T) {
		pt := PromptTemplate{
			ID:        uuid.New(),
			Name:      "system-behavior",
			Content:   "You are a helpful assistant.",
			Role:      PromptRoleSystem,
			Version:   1,
			Status:    PromptStatusActive,
			CreatedAt: time.Now(),
		}
		assert.NoError(t, pt.Validate())
	})

	t.Run("Missing ID", func(t *testing.T) {
		pt := PromptTemplate{
			ID:      uuid.Nil,
			Name:    "system-behavior",
			Role:    PromptRoleSystem,
			Version: 1,
			Status:  PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		pt := PromptTemplate{
			ID:      uuid.New(),
			Name:    "",
			Role:    PromptRoleSystem,
			Version: 1,
			Status:  PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "name is required")
	})

	t.Run("Invalid Role", func(t *testing.T) {
		pt := PromptTemplate{
			ID:      uuid.New(),
			Name:    "system-behavior",
			Role:    "invalid-role",
			Version: 1,
			Status:  PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "invalid role")
	})

	t.Run("Invalid Status", func(t *testing.T) {
		pt := PromptTemplate{
			ID:      uuid.New(),
			Name:    "system-behavior",
			Role:    PromptRoleSystem,
			Version: 1,
			Status:  "invalid-status",
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "invalid status")
	})

	t.Run("Invalid Version", func(t *testing.T) {
		pt := PromptTemplate{
			ID:      uuid.New(),
			Name:    "system-behavior",
			Role:    PromptRoleSystem,
			Version: 0,
			Status:  PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "version must be greater than zero")
	})
}

func TestPromptCollection_Validate(t *testing.T) {
	t.Run("Valid Prompt Collection", func(t *testing.T) {
		pc := PromptCollection{
			ID:          uuid.New(),
			Name:        "agent-a-prompts",
			Description: "Collection for agent A",
			Templates:   []uuid.UUID{uuid.New()},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		assert.NoError(t, pc.Validate())
	})

	t.Run("Missing ID", func(t *testing.T) {
		pc := PromptCollection{
			ID:   uuid.Nil,
			Name: "agent-a-prompts",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		pc := PromptCollection{
			ID:   uuid.New(),
			Name: "",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "name is required")
	})
}
