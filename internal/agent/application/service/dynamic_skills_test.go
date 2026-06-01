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
	t.Run("happy path: enable_skill dynamically loads procedural guidelines and brain uses them", func(t *testing.T) {
		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				if stepCount == 1 {
					// Assert the brain is presented with the available skills names/descriptions
					assert.Contains(t, request.SystemPrompt, "Available Skills")
					assert.Contains(t, request.SystemPrompt, "- Name: math")
					assert.Contains(t, request.SystemPrompt, "Description: Dynamic math instructions")

					// Step 1: Brain calls enable_skill tool to load math skill
					toolCall := map[string]any{
						"thought": "I need to calculate pool size. Let me load the math guidelines.",
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
					// Step 2: Brain should have math skill enabled, checks history for guidelines content
					assert.True(t, len(request.History) >= 2)
					
					// Assert history has the tool activation observation turn containing dynamic content
					lastTurn := request.History[len(request.History)-1]
					assert.Equal(t, "tool", lastTurn.Role)
					assert.Contains(t, lastTurn.Content, "Skill math enabled successfully")
					assert.Contains(t, lastTurn.Content, "Math Guidelines: strictly verify that 40 + 10 = 50")

					finalAnswer := map[string]any{
						"final_answer": "Verified using guidelines: the total pool size is 50.",
					}
					data, _ := json.Marshal(finalAnswer)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				return nil, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)

		// Structured propagated prompt containing instructions and skills contents
		config := service.PropagatedAgentConfig{
			Description: "Standalone test assistant",
			Directives:  "You are a precise assistant.",
			Skills: []service.Skill{
				{
					Name:        "math",
					Description: "Dynamic math instructions",
					Content:     "Math Guidelines: strictly verify that 40 + 10 = 50.",
				},
			},
		}
		systemPromptBytes, _ := json.Marshal(config)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Add 40 and 10", []model.MemoryEntry{}, string(systemPromptBytes))
		assert.NoError(t, err)
		assert.Equal(t, "Verified using guidelines: the total pool size is 50.", finalAnswer)
		assert.Equal(t, 2, stepCount)
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
		config := service.PropagatedAgentConfig{
			Description: "Standalone test assistant",
			Directives:  "You are an assistant.",
			Skills:      []service.Skill{},
		}
		systemPromptBytes, _ := json.Marshal(config)

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Activate forbidden", []model.MemoryEntry{}, string(systemPromptBytes))
		assert.NoError(t, err)
		assert.Equal(t, "I cannot load that skill.", finalAnswer)
	})

	t.Run("backward compatibility: statically registered skills work as fallback", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				if len(request.History) > 0 {
					lastTurn := request.History[len(request.History)-1]
					if lastTurn.Role == "tool" {
						assert.Contains(t, lastTurn.Content, "Static procedural guidelines content")
						finalAnswer := map[string]any{
							"final_answer": "Static skill loaded successfully.",
						}
						data, _ := json.Marshal(finalAnswer)
						return &model.BrainResponse{Content: string(data)}, nil
					}
				}

				toolCall := map[string]any{
					"tool_call": map[string]any{
						"name": "enable_skill",
						"arguments": map[string]any{
							"skill_name": "static-skill",
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		engine := service.NewCognitiveEngine(mockBrain, 5)
		engine.RegisterSkill(service.Skill{
			Name:        "static-skill",
			Description: "Static description",
			Content:     "Static procedural guidelines content.",
		})

		ctx := context.Background()
		finalAnswer, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "load static", []model.MemoryEntry{}, "You are an assistant.")
		assert.NoError(t, err)
		assert.Equal(t, "Static skill loaded successfully.", finalAnswer)
	})
}
