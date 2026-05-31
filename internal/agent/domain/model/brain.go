package model

import "errors"

// BrainRequest encapsulates parameters sent to the reasoning engine.
type BrainRequest struct {
	Prompt            string             `json:"prompt"`
	SystemPrompt      string             `json:"system_prompt,omitempty"`
	History           []MemoryEntry      `json:"history,omitempty"`
	Temperature       float64            `json:"temperature,omitempty"`
	MaxTokens         int                `json:"max_tokens,omitempty"`
	ProviderOptions   map[string]any     `json:"provider_options,omitempty"`
}

// TokenUsage tracks consumed computational units.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// BrainResponse represents a completed reasoning result.
type BrainResponse struct {
	Content          string      `json:"content"`
	Usage            TokenUsage  `json:"usage"`
	FinishReason     string      `json:"finish_reason"`
}

// BrainStreamChunk holds progressive chunks for streaming API responses.
type BrainStreamChunk struct {
	Content          string      `json:"content"`
	FinishReason     string      `json:"finish_reason,omitempty"`
}

// Validate ensures standard request constraints are met.
func (r *BrainRequest) Validate() error {
	if r.Prompt == "" {
		return errors.New("prompt must not be empty")
	}
	return nil
}
