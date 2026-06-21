package openai_test

import (
	"context"
	"encoding/json"
	"io"
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

	t.Run("should not retry on 400 Bad Request error", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "Invalid request parameters", "type": "invalid_request_error"}}`))
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

	t.Run("should parse native tool calls and convert tool history correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read request body to verify messages and tools mapping
			var body struct {
				Messages []map[string]any `json:"messages"`
				Tools    []map[string]any `json:"tools"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)

			// Assertions on the request sent to OpenAI
			// We expect 3 messages: user, assistant (with tool_calls), tool (with tool_call_id)
			// The user message must come first (chronological ordering requirement).
			if len(body.Messages) == 3 {
				// User message (first – triggered the tool call)
				userMsg := body.Messages[0]
				assert.Equal(t, "user", userMsg["role"])
				assert.Equal(t, "Next question", userMsg["content"])

				// Assistant message
				asst := body.Messages[1]
				assert.Equal(t, "assistant", asst["role"])
				assert.NotNil(t, asst["tool_calls"])

				// Tool message
				toolMsg := body.Messages[2]
				assert.Equal(t, "tool", toolMsg["role"])
				assert.Equal(t, "call_123", toolMsg["tool_call_id"])
				assert.Equal(t, "observation result", toolMsg["content"])
			}

			// We expect 1 tool definition
			if assert.Len(t, body.Tools, 1) {
				tool := body.Tools[0]
				assert.Equal(t, "function", tool["type"])
				fn := tool["function"].(map[string]any)
				assert.Equal(t, "my-tool", fn["name"])
				assert.Equal(t, "Execute tool", fn["description"])
			}

			// Respond with a tool call from model
			resp := map[string]any{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": nil,
							"tool_calls": []map[string]any{
								{
									"id":   "call_456",
									"type": "function",
									"function": map[string]any{
										"name":      "another-tool",
										"arguments": `{"val": 42}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     15,
					"completion_tokens": 20,
					"total_tokens":      35,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint: server.URL,
			APIKey:   "mock-key",
			Model:    "gpt-4o",
			Timeout:  2 * time.Second,
		})

		history := []model.MemoryEntry{
			{
				Role:    "user",
				Content: "Next question",
			},
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
			Prompt:  "Next question",
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

	t.Run("should set tool_choice to enable_skill when skills are present and no tool observation exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ToolChoice map[string]any `json:"tool_choice"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)

			// Verify tool_choice is set to {"type": "function", "function": {"name": "enable_skill"}}
			assert.Equal(t, "function", body.ToolChoice["type"])
			fn := body.ToolChoice["function"].(map[string]any)
			assert.Equal(t, "enable_skill", fn["name"])

			resp := map[string]any{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Running enable_skill",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 10,
					"total_tokens":      20,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint: server.URL,
			APIKey:   "mock-key",
			Model:    "gpt-4o",
			Timeout:  2 * time.Second,
		})

		req := model.BrainRequest{
			Prompt: "Hello",
			Tools: []model.ToolDefinition{
				{
					Name:        "enable_skill",
					Description: "Enable skill",
				},
			},
		}

		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Running enable_skill", res.Content)
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

	t.Run("should serialize system role in history correctly", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, _ := io.ReadAll(r.Body)
			var body struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.Unmarshal(bodyBytes, &body)
			
			// Verify that the system message in history was mapped to role "system"
			hasSystemMsgInHistory := false
			for _, m := range body.Messages {
				if m.Role == "system" && m.Content == "System observation" {
					hasSystemMsgInHistory = true
					break
				}
			}
			if hasSystemMsgInHistory {
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
								"content": "OK",
							},
							"finish_reason": "stop",
						},
					},
					"usage": map[string]any{
						"prompt_tokens":     10,
						"completion_tokens": 10,
						"total_tokens":      20,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": {"message": "system message not found"}}`))
			}
		}))
		defer server.Close()

		adapter := openai.NewAdapter(openai.Config{
			Endpoint: server.URL,
			APIKey:   "mock-key",
			Model:    "gpt-4o",
			Timeout:  2 * time.Second,
		})

		req := model.BrainRequest{
			Prompt: "Query",
			History: []model.MemoryEntry{
				{Role: "system", Content: "System observation"},
			},
		}

		res, err := adapter.Generate(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "OK", res.Content)
	})
}
