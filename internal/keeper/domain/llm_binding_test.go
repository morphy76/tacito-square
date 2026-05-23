package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLLMBinding_Validation(t *testing.T) {
	validBinding := LLMBinding{
		ID:                 uuid.New(),
		TenantID:           "test-tenant.com",
		Name:               "openai-gpt4o",
		Description:        "Production GPT-4o Binding",
		Provider:           ProviderOpenAI,
		APIBaseURL:         "https://api.openai.com/v1",
		APIKeySecretRef:    "openai-api-key",
		DefaultModel:       "gpt-4o",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
		TimeoutSeconds:     30,
		Status:             StatusActive,
	}

	t.Run("Valid LLM Binding", func(t *testing.T) {
		err := validBinding.Validate()
		assert.NoError(t, err)
	})

	t.Run("Missing Tenant ID", func(t *testing.T) {
		invalid := validBinding
		invalid.TenantID = ""
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant id is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		invalid := validBinding
		invalid.Name = ""
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("Invalid Provider", func(t *testing.T) {
		invalid := validBinding
		invalid.Provider = "invalid-provider"
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid provider")
	})

	t.Run("Missing API Base URL", func(t *testing.T) {
		invalid := validBinding
		invalid.APIBaseURL = ""
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api base url is required")
	})

	t.Run("Missing Default Model", func(t *testing.T) {
		invalid := validBinding
		invalid.DefaultModel = ""
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default model is required")
	})

	t.Run("Invalid Default Temperature", func(t *testing.T) {
		invalid := validBinding
		invalid.DefaultTemperature = 2.5
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")

		invalid.DefaultTemperature = -0.5
		err = invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temperature must be between 0.0 and 2.0")
	})

	t.Run("Invalid Default Max Tokens", func(t *testing.T) {
		invalid := validBinding
		invalid.DefaultMaxTokens = -10
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max tokens must be positive")
	})

	t.Run("Invalid Timeout Seconds", func(t *testing.T) {
		invalid := validBinding
		invalid.TimeoutSeconds = -5
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be positive")
	})

	t.Run("Invalid Status", func(t *testing.T) {
		invalid := validBinding
		invalid.Status = "invalid-status"
		err := invalid.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}
