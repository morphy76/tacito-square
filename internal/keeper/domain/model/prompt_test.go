package model

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
			TenantID:  "test-tenant.com",
			Name:      "system-behavior",
			Content:   "You are a helpful assistant.",
			Status:    PromptStatusActive,
			CreatedAt: time.Now(),
		}
		assert.NoError(t, pt.Validate())
	})

	t.Run("Missing Tenant ID", func(t *testing.T) {
		pt := PromptTemplate{
			ID:       uuid.New(),
			TenantID: "",
			Name:     "system-behavior",
			Status:   PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "tenant id is required")
	})

	t.Run("Missing ID", func(t *testing.T) {
		pt := PromptTemplate{
			ID:       uuid.Nil,
			TenantID: "test-tenant.com",
			Name:     "system-behavior",
			Status:   PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		pt := PromptTemplate{
			ID:       uuid.New(),
			TenantID: "test-tenant.com",
			Name:     "",
			Status:   PromptStatusActive,
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "name is required")
	})

	t.Run("Invalid Status", func(t *testing.T) {
		pt := PromptTemplate{
			ID:       uuid.New(),
			TenantID: "test-tenant.com",
			Name:     "system-behavior",
			Status:   "invalid-status",
		}
		assert.Error(t, pt.Validate())
		assert.Contains(t, pt.Validate().Error(), "invalid status")
	})
}

func TestPromptCollection_Validate(t *testing.T) {
	t.Run("Valid Prompt Collection", func(t *testing.T) {
		pc := PromptCollection{
			ID:          uuid.New(),
			TenantID:    "test-tenant.com",
			Name:        "agent-a-prompts",
			Description: "Collection for agent A",
			Templates:   []uuid.UUID{uuid.New()},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		assert.NoError(t, pc.Validate())
	})

	t.Run("Missing Tenant ID", func(t *testing.T) {
		pc := PromptCollection{
			ID:       uuid.New(),
			TenantID: "",
			Name:     "agent-a-prompts",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "tenant id is required")
	})

	t.Run("Missing ID", func(t *testing.T) {
		pc := PromptCollection{
			ID:       uuid.Nil,
			TenantID: "test-tenant.com",
			Name:     "agent-a-prompts",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		pc := PromptCollection{
			ID:       uuid.New(),
			TenantID: "test-tenant.com",
			Name:     "",
		}
		assert.Error(t, pc.Validate())
		assert.Contains(t, pc.Validate().Error(), "name is required")
	})
}
