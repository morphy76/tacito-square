package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

type tenantCtxKey struct{}
type agentCtxKey struct{}
type threadCtxKey struct{}
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

func GetThreadID(ctx context.Context) string {
	if v, ok := ctx.Value(threadCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// Skill represents a dynamic procedural knowledge source.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// PropagatedAgentConfig defines the structured keeper-agent context format.
type PropagatedAgentConfig struct {
	Description string  `json:"description"`
	Directives  string  `json:"directives"`
	Skills      []Skill `json:"skills"`
}

type parsedSkillsKey struct{}

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

type parsedResponse struct {
	Thought     string          `json:"thought"`
	ToolCall    *toolCallDetail `json:"tool_call,omitempty"`
	FinalAnswer string          `json:"final_answer,omitempty"`
}

type toolCallDetail struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

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

func (e *CognitiveEngine) WithLTM(embedder outbound.Embedder, ltm outbound.LongTermMemory) *CognitiveEngine {
	e.embedder = embedder
	e.ltm = ltm
	e.RegisterTool("recall_memory", e.handleRecallMemory)
	return e
}

func (e *CognitiveEngine) WithPublisher(publisher outbound.EventPublisher) *CognitiveEngine {
	e.publisher = publisher
	return e
}

func (e *CognitiveEngine) WithToolExecutor(mcpExecutor outbound.ToolExecutor) *CognitiveEngine {
	e.mcpExecutor = mcpExecutor
	return e
}

func (e *CognitiveEngine) WithCommunityID(communityID string) *CognitiveEngine {
	e.communityID = communityID
	return e
}

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

func (e *CognitiveEngine) handleReadLargePayload(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	if e.blobStore == nil {
		logger.Warn().Msg("read_large_payload called but blobStore is nil")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return "Error: key parameter must be a non-empty string", nil
	}

	// Read from S3 using the shared BlobStore
	reader, err := e.blobStore.Get(ctx, key)
	if err != nil {
		logger.Warn().Err(err).Str("key", key).Msg("failed to read from object storage")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}
	defer reader.Close()

	// Implement stream-buffered reading from S3 protected by maxReadSize safety limits
	limitedReader := io.LimitReader(reader, int64(e.maxReadSize))

	var sb strings.Builder
	buf := make([]byte, e.chunkSize)
	for {
		nr, er := limitedReader.Read(buf)
		if nr > 0 {
			sb.Write(buf[:nr])
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			logger.Warn().Err(er).Str("key", key).Msg("failed during buffered read from S3")
			return `{"error": "Object storage temporarily unavailable."}`, nil
		}
	}

	return sb.String(), nil
}

func (e *CognitiveEngine) handleWriteLargePayload(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	if e.blobStore == nil {
		logger.Warn().Msg("write_large_payload called but blobStore is nil")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return "Error: content parameter must be a non-empty string", nil
	}

	contentType, _ := args["content_type"].(string)
	if contentType == "" {
		contentType = "text/plain"
	}

	tenantID := GetTenantID(ctx)
	agentID := GetAgentID(ctx)
	threadID := GetThreadID(ctx)

	if tenantID == "" {
		tenantID = "default"
	}
	if agentID == "" {
		agentID = "default"
	}
	if threadID == "" {
		threadID = "default"
	}

	normalizedBucket := normalizeBucketName(tenantID)
	objectID := uuid.New().String()

	commID := e.communityID
	if commID == "" {
		commID = "default"
	}

	s3Key := fmt.Sprintf("%s/output/%s/%s/%s", commID, agentID, threadID, objectID)

	reader := strings.NewReader(content)

	ctx, span := e.tracer.Start(ctx, "cognitive_engine.write_large_payload",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("s3.bucket", normalizedBucket),
			attribute.String("s3.key", s3Key),
			attribute.Int64("s3.size_bytes", int64(len(content))),
		),
	)
	defer span.End()

	_, err := e.blobStore.Put(ctx, s3Key, reader, contentType)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Warn().Err(err).Str("key", s3Key).Msg("failed to write to object storage")
		return `{"error": "Object storage temporarily unavailable."}`, nil
	}

	ref := map[string]any{
		"_type":        "s3_reference",
		"bucket":       normalizedBucket,
		"key":          s3Key,
		"size_bytes":   int64(len(content)),
		"content_type": contentType,
	}
	refJSON, _ := json.Marshal(ref)
	return string(refJSON), nil
}

func normalizeBucketName(tenantName string) string {
	name := strings.ToLower(tenantName)
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	name = sb.String()
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
		name = strings.TrimSuffix(name, "-")
	}
	return name
}

func (e *CognitiveEngine) RegisterTool(name string, handler ToolHandler) {
	e.toolRegistry[name] = handler
}

func (e *CognitiveEngine) RegisterSkill(s Skill) {
	e.skills[s.Name] = s
}

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
		
		// If skills exist, format names and descriptions for the brain
		var skillDescs []string
		skillsMap = make(map[string]Skill)
		for _, s := range parsedConfig.Skills {
			skillDescs = append(skillDescs, fmt.Sprintf("- Name: %s\n  Description: %s", s.Name, s.Description))
			skillsMap[s.Name] = s
		}
		
		if len(skillDescs) > 0 {
			skillsList := "\n\nAvailable Skills (use 'enable_skill' tool to load their guidelines):\n" + strings.Join(skillDescs, "\n")
			activeSystemPrompt = activeSystemPrompt + skillsList
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

	if e.mcpExecutor != nil {
		mcpTools, err := e.mcpExecutor.ListAllowedTools(ctx)
		if err == nil && len(mcpTools) > 0 {
			var sb strings.Builder
			sb.WriteString("\n\nAvailable External Tools:\n")
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
				schemaJSON, _ := json.Marshal(tool.InputSchema)
				sb.WriteString(fmt.Sprintf("- Name: %s\n  Description: %s\n  Parameters: %s\n", tool.Name, tool.Description, string(schemaJSON)))
			}
			activeSystemPrompt = activeSystemPrompt + sb.String()
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

	for step := 1; step <= e.maxSteps; step++ {
		finalAnswer, shouldContinue, lastParsedThought, err := e.executeStep(
			ctx, step, tenantID, agentID, threadID,
			userQuery, activeSystemPrompt, &activeHistory,
			activeTools, logger,
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
			logger.Info().Str("final_answer", finalAnswer).Msg("ExecuteReasoningLoop completed successfully")
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
	activeTools map[string]ToolHandler,
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
	}

	resp, err := e.brain.Generate(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", false, "", err
	}

	// 2. Parse brain response
	var parsed parsedResponse
	isJSON := json.Unmarshal([]byte(resp.Content), &parsed) == nil

	if !isJSON {
		// Fallback: raw text response is immediately treated as final answer
		span.SetStatus(codes.Ok, "")
		logger.Debug().Msg("brain returned raw text; treating as final answer")
		return resp.Content, false, "", nil
	}

	// Construct Domain Model reasoning step payload
	payload := model.AgentReasoningStepPayload{
		StepIndex: step,
		Thought:   parsed.Thought,
		Timestamp: time.Now().UTC(),
	}

	if parsed.Thought != "" {
		span.AddEvent("thought", trace.WithAttributes(attribute.String("thought", parsed.Thought)))
		*activeHistory = append(*activeHistory, model.MemoryEntry{
			Role:      "assistant",
			Content:   "Thought: " + parsed.Thought,
			Timestamp: time.Now().UTC(),
		})
	}

	// Check if it's a tool call
	if parsed.ToolCall != nil && parsed.ToolCall.Name != "" {
		toolName := parsed.ToolCall.Name
		toolArgs := parsed.ToolCall.Arguments

		payload.Action = &model.ToolCallAction{
			Tool:  toolName,
			Input: toolArgs,
		}

		span.AddEvent("tool_call", trace.WithAttributes(attribute.String("tool", toolName)))

		// Output proper structured JSON stdout log for Thought + Action
		e.logStep(ctx, tenantID, agentID, threadID, payload)
		// Emit intermediate event asynchronously over NATS publisher port
		e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)

		// Register tool call to active trace history
		*activeHistory = append(*activeHistory, model.MemoryEntry{
			Role:      "assistant",
			Content:   fmt.Sprintf("Call: %s with args %v", toolName, toolArgs),
			Timestamp: time.Now().UTC(),
		})

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
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Register observation to active trace history
		*activeHistory = append(*activeHistory, model.MemoryEntry{
			Role:      "tool",
			Content:   observation,
			Timestamp: time.Now().UTC(),
		})

		// Update payload with observation, and publish again
		payload.Observation = observation
		payload.Timestamp = time.Now().UTC()

		// Output Proper structured JSON log for Action Observation
		e.logStep(ctx, tenantID, agentID, threadID, payload)
		// Emit updated intermediate NATS event
		e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)

		// Continue reasoning loop with tool observation injected
		return "", true, parsed.Thought, nil
	}

	span.SetStatus(codes.Ok, "")

	// If no tool call, log and publish thought-only step event
	e.logStep(ctx, tenantID, agentID, threadID, payload)
	e.emitStepEvent(ctx, tenantID, agentID, threadID, payload)

	// Check if final answer is present
	if parsed.FinalAnswer != "" {
		logger.Debug().Str("final_answer", parsed.FinalAnswer).Msg("parsed final answer")
		return parsed.FinalAnswer, false, parsed.Thought, nil
	}

	// If LLM returned JSON but no tool call and no final answer, default to thought or content
	if parsed.Thought != "" {
		return parsed.Thought, false, parsed.Thought, nil
	}
	return resp.Content, false, parsed.Thought, nil
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

func (e *CognitiveEngine) handleRecallMemory(ctx context.Context, args map[string]any) (string, error) {
	logger := zerolog.Ctx(ctx)

	bypassLTM := false
	if e.cfg != nil {
		bypassLTM = e.cfg.GetBool("bypass.ltm")
	}
	if bypassLTM {
		logger.Debug().Msg("recall_memory bypassed because bypass.ltm is set to true in config")
		return "No relevant memories found.", nil
	}

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

	// Try request-scoped parsed skills first
	var skill Skill
	var exists bool
	if skills, ok := ctx.Value(parsedSkillsKey{}).(map[string]Skill); ok {
		skill, exists = skills[skillName]
	}
	// Fallback to static registered engine skills
	if !exists {
		skill, exists = e.skills[skillName]
	}

	if !exists {
		logger.Warn().Str("skill", skillName).Msg("enable_skill requested for unauthorized or non-existent skill")
		return "Skill unauthorized or not found.", nil
	}

	return fmt.Sprintf("Skill %s enabled successfully. Procedural Guidelines:\n%s", skillName, skill.Content), nil
}
