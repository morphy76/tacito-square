package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/rs/zerolog"
)

// OrchestrationAction defines the structured response schema returned by the Hub's routing brain.
type OrchestrationAction struct {
	Action   string      `json:"action"` // "delegate" or "finalize"
	Spokes   []SpokeTask `json:"spokes,omitempty"`
	Response string      `json:"response,omitempty"`
}

// SpokeTask defines a single task delegated to a specific Spoke agent.
type SpokeTask struct {
	Spoke   string `json:"spoke"`
	Message string `json:"message"`
}

type Orchestrator struct {
	agentID     string
	agentName   string
	communityID string
	brain       outbound.Brain
	stateStore  outbound.OrchestrationStateStore
	lock        outbound.ThreadLock
	discovery   outbound.AgentDiscovery
	memory      outbound.ShortTermMemory
	publisher   outbound.EventPublisher
	basePrompt  string
}

func NewOrchestrator(
	agentID string,
	agentName string,
	communityID string,
	brain outbound.Brain,
	stateStore outbound.OrchestrationStateStore,
	lock outbound.ThreadLock,
	discovery outbound.AgentDiscovery,
	memory outbound.ShortTermMemory,
	publisher outbound.EventPublisher,
	basePrompt string,
) *Orchestrator {
	return &Orchestrator{
		agentID:     agentID,
		agentName:   agentName,
		communityID: communityID,
		brain:       brain,
		stateStore:  stateStore,
		lock:        lock,
		discovery:   discovery,
		memory:      memory,
		publisher:   publisher,
		basePrompt:  basePrompt,
	}
}

// ProcessUserMessage starts/restarts the orchestration flow when a user message is received.
func (o *Orchestrator) ProcessUserMessage(ctx context.Context, tenantID, threadID string, payload events.AddUserMessagePayload, correlationEventID string) error {
	logger := zerolog.Ctx(ctx).With().
		Str("thread_id", threadID).
		Str("tenant_id", tenantID).
		Str("hub_agent_id", o.agentID).
		Logger()
	ctx = logger.WithContext(ctx)

	// 1. Acquire thread lock
	locked, err := o.lock.Lock(ctx, tenantID, threadID)
	if err != nil {
		return fmt.Errorf("failed to acquire thread lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock for thread %s", threadID)
	}
	defer func() {
		if err := o.lock.Unlock(ctx, tenantID, threadID); err != nil {
			logger.Error().Err(err).Msg("failed to release thread lock")
		}
	}()

	// 2. Append user message to memory history
	userEntry := model.MemoryEntry{
		Role:      "user",
		Content:   payload.Message,
		Timestamp: time.Now().UTC(),
	}
	if err := o.memory.Append(ctx, tenantID, o.agentID, threadID, userEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append user message to short-term memory")
	}

	// 3. Discover known Spokes to compute MaxLoops dynamically at flow start time
	spokes, err := o.discovery.GetCards(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to fetch spoke cards for loop detection limit, defaulting to 5")
	}
	maxLoops := len(spokes) + 3
	if maxLoops < 3 {
		maxLoops = 5
	}

	// 4. Initialize orchestration state
	state := model.OrchestrationState{
		ThreadID:        threadID,
		CommunityID:     o.communityID,
		Status:          "idle",
		OriginalEventID: correlationEventID,
		LoopCount:       0,
		MaxLoops:        maxLoops,
	}
	if err := o.stateStore.SaveState(ctx, tenantID, threadID, state); err != nil {
		return fmt.Errorf("failed to save orchestration state: %w", err)
	}

	// 5. Run first orchestration turn
	return o.runOrchestrationTurn(ctx, tenantID, threadID, &state, payload.Message)
}

// ProcessSpokeResponse continues the orchestration flow when a subagent finishes a task.
func (o *Orchestrator) ProcessSpokeResponse(ctx context.Context, tenantID, threadID string, payload events.AgentResponsePayload) error {
	logger := zerolog.Ctx(ctx).With().
		Str("thread_id", threadID).
		Str("tenant_id", tenantID).
		Str("spoke_agent", payload.AgentName).
		Logger()
	ctx = logger.WithContext(ctx)

	// Guard: Ignore if this is the Hub's own final response
	if payload.AgentName == o.agentName {
		logger.Debug().Msg("received our own final response event, ignoring")
		return nil
	}

	// 1. Acquire thread lock
	locked, err := o.lock.Lock(ctx, tenantID, threadID)
	if err != nil {
		return fmt.Errorf("failed to acquire thread lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("could not acquire lock for thread %s", threadID)
	}
	defer func() {
		if err := o.lock.Unlock(ctx, tenantID, threadID); err != nil {
			logger.Error().Err(err).Msg("failed to release thread lock")
		}
	}()

	// 2. Load orchestration state
	state, err := o.stateStore.GetState(ctx, tenantID, threadID)
	if err != nil {
		return fmt.Errorf("failed to retrieve state: %w", err)
	}
	if state == nil || state.Status != "waiting_spoke" {
		logger.Warn().Msg("received spoke response but no active waiting orchestration state found, ignoring")
		return nil
	}

	// 3. Validate if we were waiting for this Spoke
	if _, pending := state.PendingSpokes[payload.AgentName]; !pending {
		logger.Warn().Str("spoke", payload.AgentName).Msg("received response from spoke we were not waiting for, ignoring")
		return nil
	}

	// 4. Update state: remove from pending list and append response to history
	delete(state.PendingSpokes, payload.AgentName)

	spokeEntry := model.MemoryEntry{
		Role:      "assistant",
		Content:   fmt.Sprintf("Agent %s responded: %s", payload.AgentName, payload.Response),
		Timestamp: time.Now().UTC(),
	}
	if err := o.memory.Append(ctx, tenantID, o.agentID, threadID, spokeEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append spoke response to short-term memory")
	}

	// 5. Emit progression update
	progMsg := fmt.Sprintf("Received response from %s.", payload.AgentName)
	if err := o.emitProgressionEvent(ctx, tenantID, threadID, state.OriginalEventID, progMsg); err != nil {
		logger.Warn().Err(err).Msg("failed to publish flow progression event")
	}

	// 6. Fan-in / Join check: Wait if there are other pending spokes
	if len(state.PendingSpokes) > 0 {
		logger.Info().Int("pending_count", len(state.PendingSpokes)).Msg("partial spoke response received, yielding and waiting for other concurrent spokes")
		if err := o.stateStore.SaveState(ctx, tenantID, threadID, *state); err != nil {
			return fmt.Errorf("failed to save updated state: %w", err)
		}
		return nil
	}

	// 7. All concurrent spokes finished, proceed to next orchestration turn
	return o.runOrchestrationTurn(ctx, tenantID, threadID, state, payload.Response)
}

func (o *Orchestrator) runOrchestrationTurn(ctx context.Context, tenantID, threadID string, state *model.OrchestrationState, latestInput string) error {
	logger := zerolog.Ctx(ctx)

	// Increment loop count
	state.LoopCount++

	// 1. Loop detection & force finalize
	if state.LoopCount > state.MaxLoops {
		logger.Warn().Int("loop_count", state.LoopCount).Int("max_loops", state.MaxLoops).Msg("orchestration loop limit exceeded, force-finalizing flow")
		
		// Clear orchestration state
		if err := o.stateStore.ClearState(ctx, tenantID, threadID); err != nil {
			logger.Warn().Err(err).Msg("failed to clear state")
		}

		// DO NOT send EndThread propagation so spoke memories are preserved and the thread is not terminated.

		// Fallback to the latest response received from the spokes (latestInput)
		finalResponse := latestInput
		if finalResponse == "" {
			finalResponse = "Orchestration limit exceeded without reaching a final answer."
		}

		respPayload := events.AgentResponsePayload{
			ThreadID:           threadID,
			CommunityID:        o.communityID,
			AgentName:          o.agentName,
			CorrelationEventID: state.OriginalEventID,
			Response:           finalResponse,
			Finished:           true,
		}
		
		sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
		responseEvent, err := events.NewDomainEvent(
			events.SchemaConversationalAgentResponse,
			sourceIdentity,
			tenantID,
			respPayload,
		)
		if err != nil {
			return fmt.Errorf("failed to construct fallback response event: %w", err)
		}

		eventData, err := json.Marshal(responseEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal fallback response event: %w", err)
		}

		subject := fmt.Sprintf("ts.community.%s.agent.%s.thread.%s.response", o.communityID, o.agentID, threadID)
		if err := o.publisher.Publish(ctx, subject, eventData); err != nil {
			return fmt.Errorf("failed to publish fallback response to NATS: %w", err)
		}

		return nil
	}

	// 2. Fetch conversational history from STM
	history, err := o.memory.Get(ctx, tenantID, o.agentID, threadID, 15)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to load short-term memory, using empty history")
		history = []model.MemoryEntry{}
	}

	// 3. Compile prompt detailing available specialized Spoke agents
	systemPrompt, err := o.compileSystemPrompt(ctx)
	if err != nil {
		return fmt.Errorf("failed to compile orchestrator system prompt: %w", err)
	}

	// 4. Invoke Brain (LLM) for routing turn
	brainReq := model.BrainRequest{
		Prompt:       latestInput,
		SystemPrompt: systemPrompt,
		History:      history,
	}

	resp, err := o.brain.Generate(ctx, brainReq)
	if err != nil {
		return fmt.Errorf("orchestrator brain generation failed: %w", err)
	}

	// 5. Parse action decision
	var action OrchestrationAction
	cleanedContent := resp.Content
	if strings.Contains(cleanedContent, "```json") {
		parts := strings.Split(cleanedContent, "```json")
		if len(parts) > 1 {
			cleanedContent = strings.Split(parts[1], "```")[0]
		}
	} else if strings.Contains(cleanedContent, "```") {
		parts := strings.Split(cleanedContent, "```")
		if len(parts) > 1 {
			cleanedContent = strings.Split(parts[1], "```")[0]
		}
	}
	cleanedContent = strings.TrimSpace(cleanedContent)

	if err := json.Unmarshal([]byte(cleanedContent), &action); err != nil {
		// Fallback: treat text as a final answer
		action = OrchestrationAction{
			Action:   "finalize",
			Response: resp.Content,
		}
	}

	// 6. Handle action
	switch action.Action {
	case "delegate":
		if len(action.Spokes) == 0 {
			return fmt.Errorf("brain decided to delegate but list of spokes is empty")
		}

		// Save state with map of pending spokes
		state.Status = "waiting_spoke"
		state.PendingSpokes = make(map[string]string)
		var spokeNames []string
		for _, task := range action.Spokes {
			state.PendingSpokes[task.Spoke] = task.Message
			spokeNames = append(spokeNames, task.Spoke)
		}

		if err := o.stateStore.SaveState(ctx, tenantID, threadID, *state); err != nil {
			return fmt.Errorf("failed to save waiting state: %w", err)
		}

		// Publish flow progression event: fanning out
		progMsg := fmt.Sprintf("Delegating tasks to: [%s]...", strings.Join(spokeNames, ", "))
		if err := o.emitProgressionEvent(ctx, tenantID, threadID, state.OriginalEventID, progMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to publish flow progression event")
		}

		// Publish task events to all targeted Spokes
		sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
		for _, task := range action.Spokes {
			taskPayload := events.AddUserMessagePayload{
				ThreadID:    threadID,
				CommunityID: o.communityID,
				Message:     task.Message,
			}

			taskEvent, err := events.NewDomainEvent(
				events.SchemaConversationalAddUserMessage,
				sourceIdentity,
				tenantID,
				taskPayload,
			)
			if err != nil {
				return fmt.Errorf("failed to construct task event for %s: %w", task.Spoke, err)
			}

			eventData, err := json.Marshal(taskEvent)
			if err != nil {
				return fmt.Errorf("failed to marshal task event: %w", err)
			}

			// Route to spoke by name: agent names are unique within a community,
			// and subjects are scoped by communityID (UUID), so this is globally unique.
			subject := fmt.Sprintf("ts.community.%s.agent.%s", o.communityID, task.Spoke)
			logger.Info().Str("subject", subject).Str("spoke", task.Spoke).Msg("publishing task to spoke agent")
			if err := o.publisher.Publish(ctx, subject, eventData); err != nil {
				return fmt.Errorf("failed to publish task to spoke %s: %w", task.Spoke, err)
			}
		}

		// Yield execution fanning out asynchronously
		return nil

	case "finalize":
		fallthrough
	default:
		// Clear orchestration state
		if err := o.stateStore.ClearState(ctx, tenantID, threadID); err != nil {
			logger.Warn().Err(err).Msg("failed to clear state")
		}

		// Append final response to memory history
		assistantEntry := model.MemoryEntry{
			Role:      "assistant",
			Content:   action.Response,
			Timestamp: time.Now().UTC(),
		}
		if err := o.memory.Append(ctx, tenantID, o.agentID, threadID, assistantEntry); err != nil {
			logger.Warn().Err(err).Msg("failed to append final response to short-term memory")
		}

		// Publish final AgentResponse (Finished: true)
		respPayload := events.AgentResponsePayload{
			ThreadID:           threadID,
			CommunityID:        o.communityID,
			AgentName:          o.agentName,
			CorrelationEventID: state.OriginalEventID,
			Response:           action.Response,
			Finished:           true,
		}

		sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
		responseEvent, err := events.NewDomainEvent(
			events.SchemaConversationalAgentResponse,
			sourceIdentity,
			tenantID,
			respPayload,
		)
		if err != nil {
			return fmt.Errorf("failed to construct final response event: %w", err)
		}

		eventData, err := json.Marshal(responseEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal final response event: %w", err)
		}

		subject := fmt.Sprintf("ts.community.%s.agent.%s.thread.%s.response", o.communityID, o.agentID, threadID)
		logger.Info().Str("subject", subject).Msg("publishing final orchestration response")
		if err := o.publisher.Publish(ctx, subject, eventData); err != nil {
			return fmt.Errorf("failed to publish final response to NATS: %w", err)
		}

		return nil
	}
}

func (o *Orchestrator) compileSystemPrompt(ctx context.Context) (string, error) {
	cards, err := o.discovery.GetCards(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(o.basePrompt)
	sb.WriteString("\n\nYou have access to the following specialized Spoke agents in this community:\n")
	for _, card := range cards {
		if card.Name == o.agentName {
			continue
		}
		sb.WriteString(fmt.Sprintf("- Name: %s\n", card.Name))
		if card.Description != "" {
			sb.WriteString(fmt.Sprintf("  Description: %s\n", card.Description))
		}
		if len(card.Skills) > 0 {
			var skills []string
			for _, skill := range card.Skills {
				skills = append(skills, skill.Name)
			}
			sb.WriteString(fmt.Sprintf("  Skills: %s\n", strings.Join(skills, ", ")))
		}
	}

	sb.WriteString(`
To coordinate the conversation, you must output a valid JSON response specifying your next step.
- To delegate tasks to Spoke subagents concurrently, output:
  {"action": "delegate", "spokes": [{"spoke": "<agent_name>", "message": "<task description>"}, ...] }
- If a specialized Spoke agent asks a clarifying question to the user or indicates that information/details are missing, you must immediately choose the "finalize" action and return that question directly to the user so they can reply.
- If you have completed the user request and want to finalize the response, output:
  {"action": "finalize", "response": "<final response message to the user>"}
- Do not delegate the wait state or try to delegate again if you are waiting for user input.
`)

	return sb.String(), nil
}

func (o *Orchestrator) emitProgressionEvent(ctx context.Context, tenantID, threadID, correlationEventID, message string) error {
	respPayload := events.AgentResponsePayload{
		ThreadID:           threadID,
		CommunityID:        o.communityID,
		AgentName:          o.agentName,
		CorrelationEventID: correlationEventID,
		Response:           message,
		Finished:           false, // Progression update
	}

	sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
	responseEvent, err := events.NewDomainEvent(
		events.SchemaConversationalAgentResponse,
		sourceIdentity,
		tenantID,
		respPayload,
	)
	if err != nil {
		return fmt.Errorf("failed to construct progression event: %w", err)
	}

	eventData, err := json.Marshal(responseEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal progression event: %w", err)
	}

	// Publish to standard response subject: ts.community.{community_id}.agent.{agent_id}.thread.{thread_id}.response
	subject := fmt.Sprintf("ts.community.%s.agent.%s.thread.%s.response", o.communityID, o.agentID, threadID)
	return o.publisher.Publish(ctx, subject, eventData)
}
