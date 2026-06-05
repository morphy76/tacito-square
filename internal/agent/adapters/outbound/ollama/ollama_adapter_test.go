package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/ollama"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaAdapter_Generate(t *testing.T) {
	t.Run("should parse chat completions response correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Mock Ollama chat completions response structure
			resp := map[string]any{
				"model":      "llama3",
				"created_at": "2026-05-31T09:20:00Z",
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello! I am Ollama.",
				},
				"done": true,
				"prompt_eval_count": 8,
				"eval_count":        10,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint:    server.URL,
			Model:       "llama3",
			Temperature: 0.7,
			MaxTokens:   100,
			Timeout:     2 * time.Second,
		})

		req := model.BrainRequest{Prompt: "Hi"}
		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Hello! I am Ollama.", res.Content)
		assert.Equal(t, 8, res.Usage.PromptTokens)
		assert.Equal(t, 10, res.Usage.CompletionTokens)
	})

	t.Run("should execute fallback on connection failures and trip breaker", func(t *testing.T) {
		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint:         "http://invalid-host:12345",
			FailureThreshold: 1, // Trip on first failure
			RecoveryTimeout:  10 * time.Second,
			FallbackMessage:  "ollama fallback text",
		})

		req := model.BrainRequest{Prompt: "Hi"}
		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err) // Fallback handles the error gracefully
		assert.Equal(t, "ollama fallback text", res.Content)
	})

	t.Run("should not retry on 429 Too Many Requests error", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": "Too Many Requests"}`))
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint:         server.URL,
			Model:            "llama3",
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

func TestOllamaAdapter_Embeddings(t *testing.T) {
	t.Run("should generate single embedding successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"model": "nomic-embed-text",
				"embeddings": [][]float64{
					{0.1, 0.2, 0.3},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint: server.URL,
			Model:    "nomic-embed-text",
			Timeout:  2 * time.Second,
		})

		res, err := adapter.CreateEmbedding(context.Background(), "test text")
		assert.NoError(t, err)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, res)
	})

	t.Run("should generate batch embeddings successfully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]any{
				"model": "nomic-embed-text",
				"embeddings": [][]float64{
					{0.1, 0.2},
					{0.3, 0.4},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint: server.URL,
			Model:    "nomic-embed-text",
		})

		res, err := adapter.CreateEmbeddingsBatch(context.Background(), []string{"one", "two"})
		assert.NoError(t, err)
		require.Len(t, res, 2)
		assert.Equal(t, []float32{0.1, 0.2}, res[0])
		assert.Equal(t, []float32{0.3, 0.4}, res[1])
	})

	t.Run("should fail on connection outages and propagate breaker failures", func(t *testing.T) {
		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint:         "http://invalid-host:12345",
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
			_, _ = w.Write([]byte(`{"error": "Too Many Requests"}`))
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint:         server.URL,
			Model:            "nomic-embed-text",
			Timeout:          2 * time.Second,
			FailureThreshold: 10,
		})

		_, err := adapter.CreateEmbedding(context.Background(), "test text")
		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "should only call the server once (no retries)")
	})
}
