package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/rs/zerolog"
)

type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

type tenantCtxKey struct{}
type agentCtxKey struct{}
type activeToolsKey struct{}

func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(tenantCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func GetAgentID(ctx context.Context) string {
	if v, ok := ctx.Value(agentCtxKey{}).(string); ok {
		return v
	}
	return ""
}

type CognitiveEngine struct {
	brain         outbound.Brain
	embedder      outbound.Embedder
	ltm           outbound.LongTermMemory
	toolRegistry  map[string]ToolHandler
	skillPool     map[string]map[string]ToolHandler
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
	engine := &CognitiveEngine{
		brain:        brain,
		toolRegistry: make(map[string]ToolHandler),
		skillPool:    make(map[string]map[string]ToolHandler),
		maxSteps:     maxSteps,
	}
	engine.RegisterTool("enable_skill", engine.handleEnableSkill)
	return engine
}

func (e *CognitiveEngine) WithLTM(embedder outbound.Embedder, ltm outbound.LongTermMemory) *CognitiveEngine {
	e.embedder = embedder
	e.ltm = ltm
	e.RegisterTool("recall_memory", e.handleRecallMemory)
	return e
}

func (e *CognitiveEngine) RegisterTool(name string, handler ToolHandler) {
	e.toolRegistry[name] = handler
}

func (e *CognitiveEngine) RegisterSkillCollection(name string, tools map[string]ToolHandler) {
	e.skillPool[name] = tools
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

	// Instantiate request-scoped, thread-safe active tools map
	activeTools := make(map[string]ToolHandler)
	for name, handler := range e.toolRegistry {
		activeTools[name] = handler
	}

	// Inject tenant, agent metadata and thread-scoped active tools into context
	ctx = context.WithValue(ctx, tenantCtxKey{}, tenantID)
	ctx = context.WithValue(ctx, agentCtxKey{}, agentID)
	ctx = context.WithValue(ctx, activeToolsKey{}, activeTools)

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

			// Execute the tool from the thread-local active tool map
			handler, exists := activeTools[toolName]
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

func (e *CognitiveEngine) handleRecallMemory(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	if e.embedder == nil || e.ltm == nil {
		logger.Warn().Msg("recall_memory called but long-term memory or embedder is nil")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Error: query parameter must be a non-empty string", nil
	}

	limit := 3
	if limitVal, ok := args["limit"]; ok {
		switch v := limitVal.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		}
	}

	var filter model.LTMFilter
	if catVal, ok := args["category"].(string); ok && catVal != "" {
		filter.Types = []model.LTMEntryType{model.LTMEntryType(catVal)}
	}

	tenantID := GetTenantID(ctx)
	agentID := GetAgentID(ctx)

	// 1. Generate text embedding vector
	vector, err := e.embedder.CreateEmbedding(ctx, query)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to generate embedding for recall_memory tool")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	// 2. Perform similarity search in Qdrant LTM
	threshold := float32(0.7)
	matches, err := e.ltm.Search(ctx, tenantID, agentID, vector, filter, limit, threshold)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to query Qdrant LTM for recall_memory tool")
		return `{"error": "Memory store temporarily unavailable."}`, nil
	}

	if len(matches) == 0 {
		return "No relevant memories found.", nil
	}

	// 3. Format matches
	var sb strings.Builder
	sb.WriteString("Matched context details:\n")
	for _, match := range matches {
		sb.WriteString(fmt.Sprintf("- %s\n", match.Content))
	}
	return sb.String(), nil
}

func (e *CognitiveEngine) handleEnableSkill(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	skillName, _ := args["skill_name"].(string)
	if skillName == "" {
		return "Error: skill_name parameter must be a non-empty string", nil
	}

	tools, exists := e.skillPool[skillName]
	if !exists {
		logger.Warn().Str("skill", skillName).Msg("enable_skill requested for unauthorized or non-existent skill collection")
		return "Skill unauthorized or not found.", nil
	}

	// Retrieve dynamic request-scoped active tools map from context
	active, ok := ctx.Value(activeToolsKey{}).(map[string]ToolHandler)
	if !ok {
		logger.Error().Msg("unable to retrieve request-scoped active tools map from context")
		return "Error: engine execution context mismatch.", nil
	}

	for name, handler := range tools {
		active[name] = handler
		logger.Debug().Str("tool", name).Str("skill", skillName).Msg("dynamically registered tool for thread execution")
	}

	return fmt.Sprintf("Skill %s enabled successfully.", skillName), nil
}
