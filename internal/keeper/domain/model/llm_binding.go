package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Provider represents the LLM API provider.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGroq      Provider = "groq"
	ProviderOllama    Provider = "ollama"
	ProviderCustom    Provider = "custom"
)

// BindingStatus represents the lifecycle state of the LLM binding.
type BindingStatus string

const (
	StatusActive    BindingStatus = "active"
	StatusSuspended BindingStatus = "suspended"
	StatusInactive  BindingStatus = "inactive"
)

// LLMBinding defines the connection configuration for an integrated LLM provider.
type LLMBinding struct {
	ID                 uuid.UUID     `json:"id"`
	TenantID           string        `json:"tenant_id"`
	Name               string        `json:"name"`
	Description        string        `json:"description"`
	Provider           Provider      `json:"provider"`
	APIBaseURL         string        `json:"api_base_url"`
	APIKeySecretRef    string        `json:"api_key_secret_ref,omitempty"`
	DefaultModel       string        `json:"default_model"`
	DefaultTemperature float64       `json:"default_temperature"`
	DefaultMaxTokens   int           `json:"default_max_tokens"`
	TimeoutSeconds     int           `json:"timeout_seconds"`
	Status             BindingStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Validate checks all business rules and invariants of the LLMBinding.
func (b LLMBinding) Validate() error {
	if b.ID == uuid.Nil {
		return errors.New("id is required")
	}
	if b.TenantID == "" {
		return errors.New("tenant id is required")
	}
	if b.Name == "" {
		return errors.New("name is required")
	}
	if b.Provider != ProviderOpenAI && b.Provider != ProviderAnthropic && b.Provider != ProviderGroq && b.Provider != ProviderOllama && b.Provider != ProviderCustom {
		return errors.New("invalid provider")
	}
	if b.APIBaseURL == "" {
		return errors.New("api base url is required")
	}
	if b.DefaultModel == "" {
		return errors.New("default model is required")
	}
	if b.DefaultTemperature < 0.0 || b.DefaultTemperature > 2.0 {
		return errors.New("temperature must be between 0.0 and 2.0")
	}
	if b.DefaultMaxTokens <= 0 {
		return errors.New("max tokens must be positive")
	}
	if b.TimeoutSeconds <= 0 {
		return errors.New("timeout must be positive")
	}
	if b.Status != StatusActive && b.Status != StatusSuspended && b.Status != StatusInactive {
		return errors.New("invalid status")
	}
	return nil
}
