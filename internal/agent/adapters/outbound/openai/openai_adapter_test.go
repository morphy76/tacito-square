package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/openai"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestOpenAIAdapter_Generate(t *testing.T) {
	t.Run("should parse completions response correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock OpenAI chat completions response structure
			resp := map[string]any{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Hello! I am OpenAI.",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     9,
					"completion_tokens": 12,
					"total_tokens":      21,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint:    server.URL,
			APIKey:      "mock-key",
			Model:       "gpt-4o",
			Temperature: 0.7,
			MaxTokens:   100,
			Timeout:     2 * time.Second,
		})

		req := model.BrainRequest{Prompt: "Hi"}
		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Hello! I am OpenAI.", res.Content)
		assert.Equal(t, 9, res.Usage.PromptTokens)
		assert.Equal(t, 12, res.Usage.CompletionTokens)
	})

	t.Run("should execute fallback on connection failures and trip breaker", func(t *testing.T) {
		adapter := openai.NewAdapter(openai.Config{
			Endpoint:         "http://invalid-host:12345",
			APIKey:           "mock-key",
			FailureThreshold: 1, // Trip on first failure
			RecoveryTimeout:  10 * time.Second,
			FallbackMessage:  "fallback text",
		})

		req := model.BrainRequest{Prompt: "Hi"}
		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err) // Fallback handles the error gracefully
		assert.Equal(t, "fallback text", res.Content)
	})
}
