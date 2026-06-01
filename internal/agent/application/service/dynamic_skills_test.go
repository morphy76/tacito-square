package service_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestDynamicSkills_Execution(t *testing.T) {
	t.Run("happy path: enable_skill dynamically registers tool and brain executes it on next step", func(t *testing.T) {
		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				if stepCount == 1 {
					// Step 1: Brain calls enable_skill tool to load math skill
					toolCall := map[string]any{
						"thought": "I need to calculate pool size. Let me enable math skills.",
						"tool_call": map[string]any{
							"name": "enable_skill",
							"arguments": map[string]any{
								"skill_name": "math",
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				if stepCount == 2 {
					// Step 2: Brain should have math skill enabled, calls "add" tool
					// Let's verify that the history has the tool activation success turn
					assert.True(t, len(request.History) >= 2)
					assert.Contains(t, request.History[len(request.History)-1].Content, "Skill math enabled successfully")

					toolCall := map[string]any{
						"thought": "Math skill is enabled. Let me add pool buffers.",
						"tool_call": map[string]any{
							"name": "add",
							"arguments": map[string]any{
								"a": float64(40),
								"b": float64(10),
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				// Step 3: Brain returns final answer
				finalAnswer := map[string]any{
					"final_answer": "The total pooling size is 50.",
				}
				data, _ := json.Marshal(finalAnswer)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)

		// Register a skill collection (pool of tools not active initially)
		addExecuted := false
		mathTools := map[string]service.ToolHandler{
			"add": func(ctx context.Context, args map[string]any) (string, error) {
				a := args["a"].(float64)
				b := args["b"].(float64)
				assert.Equal(t, float64(40), a)
				assert.Equal(t, float64(10), b)
				addExecuted = true
				return "Result: 50", nil
			},
		}
		engine.RegisterSkillCollection("math", mathTools)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Add 40 and 10", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "The total pooling size is 50.", finalAnswer)
		assert.True(t, addExecuted)
		assert.Equal(t, 3, stepCount)
	})

	t.Run("unauthorized skill: enable_skill returns not found error observation", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				if len(request.History) > 0 {
					lastTurn := request.History[len(request.History)-1]
					if lastTurn.Role == "tool" {
						assert.Contains(t, lastTurn.Content, "Skill unauthorized or not found")
						finalAnswer := map[string]any{
							"final_answer": "I cannot load that skill.",
						}
						data, _ := json.Marshal(finalAnswer)
						return &model.BrainResponse{Content: string(data)}, nil
					}
				}

				toolCall := map[string]any{
					"tool_call": map[string]any{
						"name": "enable_skill",
						"arguments": map[string]any{
							"skill_name": "forbidden_skill",
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)
		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Activate forbidden", []model.MemoryEntry{}, "")
		assert.NoError(t, err)
		assert.Equal(t, "I cannot load that skill.", finalAnswer)
	})
}
