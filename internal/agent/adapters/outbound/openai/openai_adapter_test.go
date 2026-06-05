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
	"github.com/stretchr/testify/require"
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

	t.Run("should not retry on 429 Too Many Requests error", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "requests", "code": "rate_limit_exceeded"}}`))
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint:         server.URL,
			APIKey:           "mock-key",
			Model:            "gpt-4o",
			Timeout:          2 * time.Second,
			FailureThreshold: 10,
		})

		req := model.BrainRequest{Prompt: "Hi"}
		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "fallback", res.FinishReason)
		assert.Equal(t, 1, callCount, "should only call the server once (no retries)")
	})
}

func TestOpenAIAdapter_Embeddings(t *testing.T) {
	t.Run("should generate single embedding successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"object": "list",
				"data": []map[string]any{
					{
						"object":    "embedding",
						"index":     0,
						"embedding": []float64{0.1, 0.2, 0.3},
					},
				},
				"model": "text-embedding-3-small",
				"usage": map[string]any{
					"prompt_tokens": 5,
					"total_tokens":  5,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint: server.URL,
			APIKey:   "mock-key",
			Model:    "text-embedding-3-small",
			Timeout:  2 * time.Second,
		})

		res, err := adapter.CreateEmbedding(context.Background(), "test text")
		assert.NoError(t, err)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, res)
	})

	t.Run("should generate batch embeddings successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"object": "list",
				"data": []map[string]any{
					{
						"object":    "embedding",
						"index":     0,
						"embedding": []float64{0.1, 0.2},
					},
					{
						"object":    "embedding",
						"index":     1,
						"embedding": []float64{0.3, 0.4},
					},
				},
				"model": "text-embedding-3-small",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint: server.URL,
			APIKey:   "mock-key",
			Model:    "text-embedding-3-small",
		})

		res, err := adapter.CreateEmbeddingsBatch(context.Background(), []string{"one", "two"})
		assert.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, []float32{0.1, 0.2}, res[0])
		assert.Equal(t, []float32{0.3, 0.4}, res[1])
	})

	t.Run("should fail on connection outages and propagate breaker failures", func(t *testing.T) {
		adapter := openai.NewAdapter(openai.Config{
			Endpoint:         "http://invalid-host:12345",
			APIKey:           "mock-key",
			FailureThreshold: 1, // Trip on first failure
		})

		_, err := adapter.CreateEmbedding(context.Background(), "test text")
		assert.Error(t, err)
	})

	t.Run("should not retry on 429 Too Many Requests error", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "requests", "code": "rate_limit_exceeded"}}`))
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint:         server.URL,
			APIKey:           "mock-key",
			Model:            "text-embedding-3-small",
			Timeout:          2 * time.Second,
			FailureThreshold: 10,
		})

		_, err := adapter.CreateEmbedding(context.Background(), "test text")
		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "should only call the server once (no retries)")
	})
}
