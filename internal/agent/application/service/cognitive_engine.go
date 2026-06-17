package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	sharedoutbound "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ToolHandler represents a function capable of executing a registered tool.
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// CognitiveEngine orchestrates the step-by-step agent reasoning and execution loops.
type CognitiveEngine struct {
	brain        outbound.Brain
	embedder     outbound.Embedder
	ltm          outbound.LongTermMemory
	publisher    outbound.EventPublisher
	mcpExecutor  outbound.ToolExecutor
	blobStore    sharedoutbound.BlobStore
	toolRegistry map[string]ToolHandler
	skills       map[string]Skill
	tracer       trace.Tracer
	communityID  string
	maxSteps     int
	maxReadSize  int
	chunkSize    int
	cfg          *viper.Viper
}

// NewCognitiveEngine creates and initializes a new instance of CognitiveEngine.
func NewCognitiveEngine(brain outbound.Brain, maxSteps int, cfg *viper.Viper) *CognitiveEngine {
	if maxSteps <= 0 {
		maxSteps = 5
	}
	engine := &CognitiveEngine{
		brain:        brain,
		toolRegistry: make(map[string]ToolHandler),
		skills:       make(map[string]Skill),
		tracer:       otel.Tracer("cognitive_engine"),
		maxSteps:     maxSteps,
		maxReadSize:  5 * 1024 * 1024,
		chunkSize:    32 * 1024,
		cfg:          cfg,
	}
	engine.RegisterTool("enable_skill", engine.handleEnableSkill)
	return engine
}

// WithLTM configures the long-term memory backend and registers the recall_memory tool.
func (e *CognitiveEngine) WithLTM(embedder outbound.Embedder, ltm outbound.LongTermMemory) *CognitiveEngine {
	e.embedder = embedder
	e.ltm = ltm
	e.RegisterTool("recall_memory", e.handleRecallMemory)
	return e
}

// WithPublisher configures the event publishing channel for streaming steps.
func (e *CognitiveEngine) WithPublisher(publisher outbound.EventPublisher) *CognitiveEngine {
	e.publisher = publisher
	return e
}

// WithToolExecutor configures the MCP tool execution adapter.
func (e *CognitiveEngine) WithToolExecutor(mcpExecutor outbound.ToolExecutor) *CognitiveEngine {
	e.mcpExecutor = mcpExecutor
	return e
}

// WithCommunityID sets the community scope identifier.
func (e *CognitiveEngine) WithCommunityID(communityID string) *CognitiveEngine {
	e.communityID = communityID
	return e
}

// WithBlobStore configures S3/Blob storage bindings and registers payload storage tools.
func (e *CognitiveEngine) WithBlobStore(blobStore sharedoutbound.BlobStore, maxReadSize, chunkSize int) *CognitiveEngine {
	e.blobStore = blobStore
	if maxReadSize > 0 {
		e.maxReadSize = maxReadSize
	}
	if chunkSize > 0 {
		e.chunkSize = chunkSize
	}
	e.RegisterTool("read_large_payload", e.handleReadLargePayload)
	e.RegisterTool("write_large_payload", e.handleWriteLargePayload)
	return e
}

// RegisterTool binds a named tool to its execution handler.
func (e *CognitiveEngine) RegisterTool(name string, handler ToolHandler) {
	e.toolRegistry[name] = handler
}

// RegisterSkill registers a static procedural skill definition.
func (e *CognitiveEngine) RegisterSkill(s Skill) {
	e.skills[s.Name] = s
}

// ExecuteReasoningLoop runs the main step-by-step reasoning loop to produce an answer.
func (e *CognitiveEngine) ExecuteReasoningLoop(
	ctx context.Context,
	tenantID, agentID, threadID string,
	userQuery string,
	history []model.MemoryEntry,
	systemPrompt string,
) (string, error) {
	ctx, span := e.tracer.Start(ctx, "cognitive_engine.loop",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	logger.Debug().Msg("entering ExecuteReasoningLoop")

	// Try to parse systemPrompt as a structured JSON PropagatedAgentConfig
	var parsedConfig PropagatedAgentConfig
	isStructured := json.Unmarshal([]byte(systemPrompt), &parsedConfig) == nil

	var activeSystemPrompt string
	var skillsMap map[string]Skill

	if isStructured {
		activeSystemPrompt = parsedConfig.Directives
		skillsMap = make(map[string]Skill)
		for _, s := range parsedConfig.Skills {
			skillsMap[s.Name] = s
		}
	} else {
		// Fallback for raw text prompt templates
		activeSystemPrompt = systemPrompt
	}

	// Instantiate request-scoped, thread-safe active tools map
	activeTools := make(map[string]ToolHandler)
	for name, handler := range e.toolRegistry {
		activeTools[name] = handler
	}

	// Define built-in tool definitions
	recallMemoryDef := model.ToolDefinition{
		Name:        "recall_memory",
		Description: "Recall relevant facts, declarations, and details from the agent's long-term memory.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to match against memories.",
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Optional category filter for semantic retrieval.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Optional limit on the number of recalled memories.",
				},
			},
			"required": []any{"query"},
		},
	}

	readPayloadDef := model.ToolDefinition{
		Name:        "read_large_payload",
		Description: "Read a large payload or file from S3 object storage.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "The unique S3 object key identifying the file.",
				},
			},
			"required": []any{"key"},
		},
	}

	writePayloadDef := model.ToolDefinition{
		Name:        "write_large_payload",
		Description: "Write a large payload or file to S3 object storage.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "The text content or payload to write to S3.",
				},
				"content_type": map[string]any{
					"type":        "string",
					"description": "Optional MIME content type descriptor.",
				},
			},
			"required": []any{"content"},
		},
	}

	var toolsToExpose []model.ToolDefinition

	if e.ltm != nil && e.embedder != nil {
		toolsToExpose = append(toolsToExpose, recallMemoryDef)
	}
	if e.blobStore != nil {
		toolsToExpose = append(toolsToExpose, readPayloadDef)
		toolsToExpose = append(toolsToExpose, writePayloadDef)
	}

	// Dynamic enum population and description formatting for enable_skill based on authorized skill names
	var skillNames []any
	var skillDescriptions []string
	seenSkills := make(map[string]bool)
	if isStructured {
		for _, s := range parsedConfig.Skills {
			if !seenSkills[s.Name] {
				seenSkills[s.Name] = true
				skillNames = append(skillNames, s.Name)
				desc := s.Description
				if desc == "" {
					desc = "No description provided"
				}
				skillDescriptions = append(skillDescriptions, s.Name+": "+desc)
			}
		}
	}
	for name, s := range e.skills {
		if !seenSkills[name] {
			seenSkills[name] = true
			skillNames = append(skillNames, name)
			desc := s.Description
			if desc == "" {
				desc = "No description provided"
			}
			skillDescriptions = append(skillDescriptions, name+": "+desc)
		}
	}

	if len(skillNames) > 0 {
		paramDescription := "The name of the skill to enable. Available options:\n" + strings.Join(skillDescriptions, "\n")
		enableSkillDef := model.ToolDefinition{
			Name:        "enable_skill",
			Description: "Enable a specific skill to load its guidelines and instructions when needed for a task.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"skill_name": map[string]any{
						"type":        "string",
						"description": paramDescription,
						"enum":        skillNames,
					},
				},
				"required": []any{"skill_name"},
			},
		}
		toolsToExpose = append(toolsToExpose, enableSkillDef)
	}

	if e.mcpExecutor != nil {
		mcpTools, err := e.mcpExecutor.ListAllowedTools(ctx)
		if err == nil && len(mcpTools) > 0 {
			for _, tool := range mcpTools {
				tName := tool.Name
				// Safety check: protect built-in tools against hijack
				if _, exists := activeTools[tName]; exists {
					continue
				}

				activeTools[tName] = func(ctx context.Context, args map[string]any) (string, error) {
					start := time.Now()

					resp, err := e.mcpExecutor.Execute(ctx, tName, args)

					duration := time.Since(start).Seconds()

					statusStr := "success"
					if err != nil || strings.Contains(resp, `"error":`) {
						statusStr = "error"
					}

					attrs := metric.WithAttributes(
						attribute.String("tool", tName),
						attribute.String("status", statusStr),
					)

					observability.AgentMCPRequestsTotal.Add(ctx, 1, attrs)
					observability.AgentMCPRequestDuration.Record(ctx, duration, attrs)

					return resp, err
				}

				toolsToExpose = append(toolsToExpose, tool)
			}
		}
	}

	// Inject tenant, agent metadata and thread-scoped active tools/parsed skills into context
	ctx = context.WithValue(ctx, tenantCtxKey{}, tenantID)
	ctx = context.WithValue(ctx, agentCtxKey{}, agentID)
	ctx = context.WithValue(ctx, threadCtxKey{}, threadID)
	ctx = context.WithValue(ctx, activeToolsKey{}, activeTools)
	if isStructured {
		ctx = context.WithValue(ctx, parsedSkillsKey{}, skillsMap)
	}

	// Ephemeral trace context is represented as virtual conversation turns appended to history context
	activeHistory := make([]model.MemoryEntry, len(history))
	copy(activeHistory, history)

	var lastThought string
	userQueryInHistory := false

	for step := 1; step <= e.maxSteps; step++ {
		finalAnswer, shouldContinue, lastParsedThought, err := e.executeStep(
			ctx, step, tenantID, agentID, threadID,
			userQuery, activeSystemPrompt, &activeHistory,
			&userQueryInHistory,
			activeTools, toolsToExpose, logger,
		)
		if lastParsedThought != "" {
			lastThought = lastParsedThought
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Error().Err(err).Int("step", step).Msg("LLM generation failed in reasoning loop")
			return "", err
		}
		if !shouldContinue {
			span.SetStatus(codes.Ok, "")
			logger.Info().Msg("ExecuteReasoningLoop completed successfully")
			logger.Trace().Str("final_answer", finalAnswer).Msg("final answer content")
			return finalAnswer, nil
		}
	}

	logger.Warn().Int("max_steps", e.maxSteps).Msg("exceeded maximum reasoning steps limit without yielding final answer")
	span.AddEvent("safeguard_limit_reached", trace.WithAttributes(
		attribute.Int("max_steps", e.maxSteps),
		attribute.String("message", "exceeded maximum reasoning steps limit without yielding final answer"),
	))
	span.SetStatus(codes.Ok, "")

	if lastThought != "" {
		return "Thought: " + lastThought, nil
	}
	return "Error: reasoning steps limit exceeded", nil
}

func (e *CognitiveEngine) executeStep(
	ctx context.Context,
	step int,
	tenantID, agentID, threadID string,
	userQuery string,
	systemPrompt string,
	activeHistory *[]model.MemoryEntry,
	userQueryInHistory *bool,
	activeTools map[string]ToolHandler,
	toolsToExpose []model.ToolDefinition,
	logger zerolog.Logger,
) (string, bool, string, error) {
	// Start OTel sub-span for this granular reasoning step
	ctx, span := e.tracer.Start(ctx, "cognitive_engine.step",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
			attribute.Int("reasoning_step_index", step),
		),
	)
	defer span.End()

	logger.Debug().Int("step", step).Msg("starting reasoning step")

	// 1. Generate LLM turn
	req := model.BrainRequest{
		Prompt:       userQuery,
		SystemPrompt: systemPrompt,
		History:      *activeHistory,
		Tools:        toolsToExpose,
	}

	resp, err := e.brain.Generate(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", false, "", err
	}

	// Extract content, tool calls, and thoughts from the brain response
	var content string = resp.Content
	var toolCalls []model.ToolCall = resp.ToolCalls
	var thought string = resp.Content

	// Graceful parsing fallback for ReAct JSON format mocks/compatibility
	var parsed parsedResponse
	if len(toolCalls) == 0 && json.Unmarshal([]byte(resp.Content), &parsed) == nil {
		if parsed.Thought != "" {
			thought = parsed.Thought
		}
		if parsed.ToolCall != nil && parsed.ToolCall.Name != "" {
			toolCalls = []model.ToolCall{
				{
					ID:        "call_" + uuid.New().String(),
					Name:      parsed.ToolCall.Name,
					Arguments: parsed.ToolCall.Arguments,
				},
			}
			content = parsed.Thought
		} else if parsed.FinalAnswer != "" {
			content = parsed.FinalAnswer
		}
	}

	if thought != "" {
		span.AddEvent("thought", trace.WithAttributes(attribute.String("thought", thought)))
	}

	// Check if it's a tool call turn
	if len(toolCalls) > 0 {
		// Ensure the user query precedes tool call entries in history.
		// LLM APIs require strict chronological ordering: user → assistant → tool.
		if !*userQueryInHistory {
			*activeHistory = append(*activeHistory, model.MemoryEntry{
				Role:      "user",
				Content:   userQuery,
				Timestamp: time.Now().UTC(),
			})
			*userQueryInHistory = true
		}

		// Register tool calls metadata to active history
		tCallsBytes, _ := json.Marshal(toolCalls)
		*activeHistory = append(*activeHistory, model.MemoryEntry{
			Role:      "assistant",
			Content:   content,
			Timestamp: time.Now().UTC(),
			Metadata: map[string]string{
				"tool_calls": string(tCallsBytes),
			},
		})

		for _, tc := range toolCalls {
			toolName := tc.Name
			toolArgs := tc.Arguments

			payload := model.AgentReasoningStepPayload{
				StepIndex: step,
				Thought:   thought,
				Action: &model.ToolCallAction{
					Tool:  toolName,
					Input: toolArgs,
				},
				Timestamp: time.Now().UTC(),
			}

			span.AddEvent("tool_call", trace.WithAttributes(attribute.String("tool", toolName)))

			// Output proper structured JSON stdout log for Thought + Action
			e.logStep(ctx, tenantID, agentID, threadID, payload)
			// Emit intermediate event asynchronously over NATS publisher port
			e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)

			// Execute the tool from the thread-local active tool map
			handler, exists := activeTools[toolName]
			var observation string
			var toolErr error
			if !exists {
				toolErr = fmt.Errorf("tool %s is not registered or allowed", toolName)
				observation = fmt.Sprintf("Error: tool %s is not registered or allowed", toolName)
				logger.Warn().Str("tool", toolName).Msg("tool execution requested but tool is not registered")
			} else {
				obs, err := handler(ctx, toolArgs)
				if err != nil {
					toolErr = err
					observation = fmt.Sprintf("Error: tool execution failed: %v", err)
					logger.Warn().Err(err).Str("tool", toolName).Msg("tool execution returned error")
				} else {
					observation = obs
				}
			}

			if toolErr != nil {
				span.RecordError(toolErr)
				span.SetStatus(codes.Error, toolErr.Error())
			}

			// Register observation to active history with tool metadata
			*activeHistory = append(*activeHistory, model.MemoryEntry{
				Role:      "tool",
				Content:   observation,
				Timestamp: time.Now().UTC(),
				Metadata: map[string]string{
					"tool_call_id": tc.ID,
					"tool_name":    toolName,
				},
			})

			// Update payload with observation, and publish again
			payload.Observation = observation
			payload.Timestamp = time.Now().UTC()

			// Output Proper structured JSON log for Action Observation
			e.logStep(ctx, tenantID, agentID, threadID, payload)
			// Emit updated intermediate NATS event
			e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)
		}

		// Continue reasoning loop with tool observations injected
		return "", true, thought, nil
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Msg("brain returned final response (no tool calls)")

	// Construct Domain Model reasoning step payload for final response
	payload := model.AgentReasoningStepPayload{
		StepIndex: step,
		Thought:   thought,
		Timestamp: time.Now().UTC(),
	}
	e.logStep(ctx, tenantID, agentID, threadID, payload)
	e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)

	return content, false, thought, nil
}

func (e *CognitiveEngine) logStep(ctx context.Context, tenantID, agentID, threadID string, payload model.AgentReasoningStepPayload) {
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Int("reasoning_step_index", payload.StepIndex).
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Str("thought", payload.Thought).
		Interface("action", payload.Action).
		Str("observation", payload.Observation).
		Msg("reasoning loop step executed")
}

func (e *CognitiveEngine) emitStepEvent(ctx context.Context, tenantID, agentID, threadID string, payload model.AgentReasoningStepPayload) {
	if e.publisher == nil {
		return
	}

	logger := zerolog.Ctx(ctx)
	subject := fmt.Sprintf("ts.tenant.%s.agent.%s.thread.%s.reasoning", tenantID, agentID, threadID)
	data, err := json.Marshal(payload)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal intermediate reasoning step payload")
		return
	}

	err = e.publisher.Publish(ctx, subject, data)
	if err != nil {
		logger.Warn().Err(err).Str("subject", subject).Msg("failed to publish intermediate reasoning event over NATS publisher port")
	}
}
