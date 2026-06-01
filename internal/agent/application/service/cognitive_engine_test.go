package service_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

// MockToolExecutor is a mock tool handler function for testing.
type MockToolExecutor func(ctx context.Context, args map[string]any) (string, error)

func TestCognitiveEngine_ExecuteReasoningLoop(t *testing.T) {
	t.Run("happy path: react thought loop completes with tool call and final answer", func(t *testing.T) {
		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				if stepCount == 1 {
					// Turn 1: brain decides to call recall_memory tool
					toolCall := map[string]any{
						"thought": "I need to recall database pooling config.",
						"tool_call": map[string]any{
							"name": "recall_memory",
							"arguments": map[string]any{
								"query": "database pooling",
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{
						Content:      string(data),
						FinishReason: "stop",
					}, nil
				}

				// Turn 2: brain has the tool observation, now returns final answer
				finalAnswer := map[string]any{
					"thought":      "The observation shows database pooling is 50. I can answer now.",
					"final_answer": "The database pooling size is configured to 50.",
				}
				data, _ := json.Marshal(finalAnswer)
				return &model.BrainResponse{
					Content:      string(data),
					FinishReason: "stop",
				}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)

		// Register mock tool executor for recall_memory
		toolExecuted := false
		engine.RegisterTool("recall_memory", func(ctx context.Context, args map[string]any) (string, error) {
			assert.Equal(t, "database pooling", args["query"])
			toolExecuted = true
			return "Result: database pool size = 50", nil
		})

		ctx := context.Background()
		finalResp, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "What is the db pooling size?", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "The database pooling size is configured to 50.", finalResp)
		assert.True(t, toolExecuted)
		assert.Equal(t, 2, stepCount)
	})

	t.Run("limits: loop halts and returns best response when max steps exceeded", func(t *testing.T) {
		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				// Brain keeps calling tool forever
				toolCall := map[string]any{
					"thought": "Looping thought...",
					"tool_call": map[string]any{
						"name":      "infinite_tool",
						"arguments": map[string]any{},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{
					Content:      string(data),
					FinishReason: "stop",
				}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 3)
		engine.RegisterTool("infinite_tool", func(ctx context.Context, args map[string]any) (string, error) {
			return "infinite tool output", nil
		})

		ctx := context.Background()
		finalResp, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Infinite query", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		// Should return the best available response (e.g. the last thought or content)
		assert.Contains(t, finalResp, "Looping thought...")
		assert.Equal(t, 3, stepCount)
	})

	t.Run("fallback: raw text output immediately treated as final answer", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				return &model.BrainResponse{
					Content:      "This is a direct final answer without JSON formatting.",
					FinishReason: "stop",
				}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)
		ctx := context.Background()
		finalResp, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Direct query", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "This is a direct final answer without JSON formatting.", finalResp)
	})
}
