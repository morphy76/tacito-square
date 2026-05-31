package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
)

type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

type CognitiveEngine struct {
	brain         outbound.Brain
	toolRegistry  map[string]ToolHandler
	maxSteps      int
}

type parsedResponse struct {
	Thought     string          `json:"thought"`
	ToolCall    *toolCallDetail `json:"tool_call,omitempty"`
	FinalAnswer string          `json:"final_answer,omitempty"`
}

type toolCallDetail struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func NewCognitiveEngine(brain outbound.Brain) *CognitiveEngine {
	maxSteps := 5
	if envVal := os.Getenv("TS_AGENT_MAX_REASONING_STEPS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			maxSteps = val
		}
	}
	return &CognitiveEngine{
		brain:        brain,
		toolRegistry: make(map[string]ToolHandler),
		maxSteps:     maxSteps,
	}
}

func (e *CognitiveEngine) RegisterTool(name string, handler ToolHandler) {
	e.toolRegistry[name] = handler
}

func (e *CognitiveEngine) ExecuteReasoningLoop(
	ctx context.Context,
	tenantID, agentID, threadID string,
	userQuery string,
	history []model.MemoryEntry,
	systemPrompt string,
) (string, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	// Ephemeral trace context is represented as virtual conversation turns appended to history context
	activeHistory := make([]model.MemoryEntry, len(history))
	copy(activeHistory, history)

	var lastThought string

	for step := 1; step <= e.maxSteps; step++ {
		logger.Debug().Int("step", step).Msg("starting reasoning step")

		// 1. Generate LLM turn
		req := model.BrainRequest{
			Prompt:       userQuery,
			SystemPrompt: systemPrompt,
			History:      activeHistory,
		}

		resp, err := e.brain.Generate(ctx, req)
		if err != nil {
			logger.Error().Err(err).Int("step", step).Msg("LLM generation failed in reasoning loop")
			return "", err
		}

		// 2. Parse brain response
		var parsed parsedResponse
		isJSON := json.Unmarshal([]byte(resp.Content), &parsed) == nil

		if !isJSON {
			// Fallback: raw text response is immediately treated as final answer
			logger.Debug().Msg("brain returned raw text; treating as final answer")
			return resp.Content, nil
		}

		if parsed.Thought != "" {
			lastThought = parsed.Thought
			logger.Debug().Str("thought", parsed.Thought).Msg("parsed brain thought")
			activeHistory = append(activeHistory, model.MemoryEntry{
				Role:      "assistant",
				Content:   "Thought: " + parsed.Thought,
				Timestamp: time.Now().UTC(),
			})
		}

		// Check if it's a tool call
		if parsed.ToolCall != nil && parsed.ToolCall.Name != "" {
			toolName := parsed.ToolCall.Name
			toolArgs := parsed.ToolCall.Arguments
			logger.Info().Str("tool", toolName).Interface("args", toolArgs).Msg("parsed tool call execution request")

			// Register tool call to active trace history
			activeHistory = append(activeHistory, model.MemoryEntry{
				Role:      "assistant",
				Content:   fmt.Sprintf("Call: %s with args %v", toolName, toolArgs),
				Timestamp: time.Now().UTC(),
			})

			// Execute the tool if registered
			handler, exists := e.toolRegistry[toolName]
			var observation string
			if !exists {
				observation = fmt.Sprintf("Error: tool %s is not registered or allowed", toolName)
				logger.Warn().Str("tool", toolName).Msg("tool execution requested but tool is not registered")
			} else {
				obs, err := handler(ctx, toolArgs)
				if err != nil {
					observation = fmt.Sprintf("Error: tool execution failed: %v", err)
					logger.Warn().Err(err).Str("tool", toolName).Msg("tool execution returned error")
				} else {
					observation = obs
				}
			}

			logger.Debug().Str("tool", toolName).Str("observation", observation).Msg("tool execution finished")

			// Register observation to active trace history
			activeHistory = append(activeHistory, model.MemoryEntry{
				Role:      "tool",
				Content:   observation,
				Timestamp: time.Now().UTC(),
			})

			// Continue reasoning loop with tool observation injected
			continue
		}

		// Check if final answer is present
		if parsed.FinalAnswer != "" {
			logger.Debug().Str("final_answer", parsed.FinalAnswer).Msg("parsed final answer")
			return parsed.FinalAnswer, nil
		}

		// If LLM returned JSON but no tool call and no final answer, default to thought or content
		if parsed.Thought != "" {
			return parsed.Thought, nil
		}
		return resp.Content, nil
	}

	logger.Warn().Int("max_steps", e.maxSteps).Msg("exceeded maximum reasoning steps limit without yielding final answer")
	if lastThought != "" {
		return "Thought: " + lastThought, nil
	}
	return "Error: reasoning steps limit exceeded", nil
}
