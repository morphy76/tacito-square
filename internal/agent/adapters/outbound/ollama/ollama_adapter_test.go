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

	t.Run("should not retry on 400 Bad Request error", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Bad Request"}`))
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

	t.Run("should parse native tool calls and convert tool history correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Assert route
			assert.Equal(t, "/api/chat", r.URL.Path)

			// Read request body to verify messages and tools mapping
			var body struct {
				Model    string           `json:"model"`
				Messages []map[string]any `json:"messages"`
				Tools    []map[string]any `json:"tools"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)

			// Assertions on history conversion
			if len(body.Messages) == 3 {
				// Assistant message with tool calls
				asst := body.Messages[0]
				assert.Equal(t, "assistant", asst["role"])
				assert.NotNil(t, asst["tool_calls"])

				// Tool message with ToolCallID and ToolName
				toolMsg := body.Messages[1]
				assert.Equal(t, "tool", toolMsg["role"])
				assert.Equal(t, "call_123", toolMsg["tool_call_id"])
				assert.Equal(t, "my-tool", toolMsg["tool_name"])
				assert.Equal(t, "observation result", toolMsg["content"])

				// User prompt
				userMsg := body.Messages[2]
				assert.Equal(t, "user", userMsg["role"])
				assert.Equal(t, "Next query", userMsg["content"])
			}

			// Assertions on tools mapping
			if assert.Len(t, body.Tools, 1) {
				assert.Equal(t, "function", body.Tools[0]["type"])
				fn := body.Tools[0]["function"].(map[string]any)
				assert.Equal(t, "my-tool", fn["name"])
				assert.Equal(t, "Execute tool", fn["description"])
			}

			// Respond with a tool call from the model
			resp := map[string]any{
				"model":      "llama3",
				"created_at": "2026-05-31T09:20:00Z",
				"message": map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []map[string]any{
						{
							"id":   "call_456",
							"type": "function",
							"function": map[string]any{
								"name": "another-tool",
								"arguments": map[string]any{
									"val": float64(42),
								},
							},
						},
					},
				},
				"done":              true,
				"prompt_eval_count": 15,
				"eval_count":        20,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := ollama.NewAdapter(ollama.Config{
			Endpoint: server.URL,
			Model:    "llama3",
			Timeout:  2 * time.Second,
		})

		history := []model.MemoryEntry{
			{
				Role:    "assistant",
				Content: "Let me call the tool.",
				Metadata: map[string]string{
					"tool_calls": `[{"id": "call_123", "name": "my-tool", "arguments": {"param": "val"}}]`,
				},
			},
			{
				Role:    "tool",
				Content: "observation result",
				Metadata: map[string]string{
					"tool_call_id": "call_123",
					"tool_name":    "my-tool",
				},
			},
		}

		tools := []model.ToolDefinition{
			{
				Name:        "my-tool",
				Description: "Execute tool",
				InputSchema: map[string]any{
					"type": "object",
				},
			},
		}

		req := model.BrainRequest{
			Prompt:  "Next query",
			History: history,
			Tools:   tools,
		}

		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "tool_calls", res.FinishReason)
		require.Len(t, res.ToolCalls, 1)
		assert.Equal(t, "call_456", res.ToolCalls[0].ID)
		assert.Equal(t, "another-tool", res.ToolCalls[0].Name)
		assert.Equal(t, float64(42), res.ToolCalls[0].Arguments["val"])
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

	t.Run("should map chat model to nomic-embed-text", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "nomic-embed-text", body.Model)

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
			Model:    "llama3",
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
