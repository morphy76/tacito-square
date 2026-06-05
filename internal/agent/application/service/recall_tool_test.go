package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestRecallMemoryTool_Execution(t *testing.T) {
	t.Run("happy path: recall memory tool fetches embeddings and searches LTM", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// Assert that on the second turn, the tool observation is injected into request history
				if len(request.History) > 0 {
					lastTurn := request.History[len(request.History)-1]
					if lastTurn.Role == "tool" {
						assert.Contains(t, lastTurn.Content, "Matched context details")
						finalAnswer := map[string]any{
							"final_answer": "According to memory, pool size is 50.",
						}
						data, _ := json.Marshal(finalAnswer)
						return &model.BrainResponse{Content: string(data)}, nil
					}
				}

				// Turn 1: brain requests recall
				toolCall := map[string]any{
					"thought": "Let me search my memories.",
					"tool_call": map[string]any{
						"name": "recall_memory",
						"arguments": map[string]any{
							"query": "db config",
							"limit": float64(2),
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		embeddingCalled := false
		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				assert.Equal(t, "db config", text)
				embeddingCalled = true
				return []float32{0.5, 0.6, 0.7}, nil
			},
		}

		ltmCalled := false
		mockLTM := &MockLongTermMemory{
			SearchFunc: func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
				assert.Equal(t, "tenant-1", tenantID)
				assert.Equal(t, "agent-1", agentID)
				assert.Equal(t, []float32{0.5, 0.6, 0.7}, vector)
				assert.Equal(t, 2, limit)
				ltmCalled = true
				return []model.LTMEntry{
					{ID: "entry-1", Content: "Matched context details", Score: 0.95},
				}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5).WithLTM(mockEmbed, mockLTM)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Find config info", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "According to memory, pool size is 50.", finalAnswer)
		assert.True(t, embeddingCalled)
		assert.True(t, ltmCalled)
	})

	t.Run("error handling: embedding generation failure returns graceful fallback observation", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				if len(request.History) > 0 {
					lastTurn := request.History[len(request.History)-1]
					if lastTurn.Role == "tool" {
						assert.Contains(t, lastTurn.Content, "Memory store temporarily unavailable")
						finalAnswer := map[string]any{
							"final_answer": "I could not retrieve memories, but I will answer best I can.",
						}
						data, _ := json.Marshal(finalAnswer)
						return &model.BrainResponse{Content: string(data)}, nil
					}
				}

				toolCall := map[string]any{
					"tool_call": map[string]any{
						"name": "recall_memory",
						"arguments": map[string]any{
							"query": "broken info",
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				return nil, errors.New("outage")
			},
		}
		mockLTM := &MockLongTermMemory{}

		engine := service.NewCognitiveEngine(mockBrain, 5).WithLTM(mockEmbed, mockLTM)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Broken query", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "I could not retrieve memories, but I will answer best I can.", finalAnswer)
	})

	t.Run("bypass path: TS_AGENT_BYPASS_LTM is set", func(t *testing.T) {
		t.Setenv("TS_AGENT_BYPASS_LTM", "true")

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				if len(request.History) > 0 {
					lastTurn := request.History[len(request.History)-1]
					if lastTurn.Role == "tool" {
						assert.Contains(t, lastTurn.Content, "No relevant memories found.")
						finalAnswer := map[string]any{
							"final_answer": "No memories because it was bypassed.",
						}
						data, _ := json.Marshal(finalAnswer)
						return &model.BrainResponse{Content: string(data)}, nil
					}
				}

				toolCall := map[string]any{
					"tool_call": map[string]any{
						"name": "recall_memory",
						"arguments": map[string]any{
							"query": "some info",
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				t.Error("Embedder should not be called when LTM is bypassed")
				return nil, errors.New("should not be called")
			},
		}
		mockLTM := &MockLongTermMemory{
			SearchFunc: func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
				t.Error("LTM search should not be called when LTM is bypassed")
				return nil, errors.New("should not be called")
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5).WithLTM(mockEmbed, mockLTM)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Query", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "No memories because it was bypassed.", finalAnswer)
	})
}
