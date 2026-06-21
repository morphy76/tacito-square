package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
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
		Status:          model.StatusIdle,
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
	if state == nil || state.Status != model.StatusWaitingSpoke {
		logger.Warn().Msg("received spoke response but no active waiting orchestration state found, ignoring")
		return nil
	}

	// 3. Validate if we were waiting for this Spoke
	var matchedSpokeKey string
	var originalMsg string
	var pending bool
	for k, v := range state.PendingSpokes {
		if strings.EqualFold(k, payload.AgentName) {
			matchedSpokeKey = k
			originalMsg = v
			pending = true
			break
		}
	}
	if !pending {
		logger.Warn().Str("spoke", payload.AgentName).Msg("received response from spoke we were not waiting for, ignoring")
		return nil
	}

	// 4. Handoff Detection & Parsing
	type HandoffSuggestion struct {
		Action string `json:"action"`
		Target string `json:"target"`
		Reason string `json:"reason"`
	}

	var handoff HandoffSuggestion
	isHandoff := false
	cleanedResponse := payload.Response
	if strings.Contains(cleanedResponse, `"suggest_handoff"`) {
		cleanedResponse = CleanAndExtractJSON(cleanedResponse)
		if err := json.Unmarshal([]byte(cleanedResponse), &handoff); err == nil && handoff.Action == "suggest_handoff" && handoff.Target != "" {
			isHandoff = true
		}
	}

	// 5. Target Validation
	var targetCard *agentcard.AgentCard
	if isHandoff {
		cards, err := o.discovery.GetCards(ctx)
		if err == nil {
			for _, card := range cards {
				if strings.EqualFold(card.Name, handoff.Target) {
					targetCard = card
					handoff.Target = card.Name // Normalize target to official name
					break
				}
			}
		}
	}

	// Remove delegating spoke from pending list
	delete(state.PendingSpokes, matchedSpokeKey)

	if targetCard != nil {
		// Handoff Execution
		logger.Info().Str("target_spoke", handoff.Target).Msg("executing coordinated handoff delegation")

		// 1. Hub Memory Logging
		observation := fmt.Sprintf("[Observation] Spoke Agent '%s' suggested handoff to '%s' because: %s", matchedSpokeKey, handoff.Target, handoff.Reason)
		spokeEntry := model.MemoryEntry{
			Role:      "system",
			Content:   observation,
			Timestamp: time.Now().UTC(),
		}
		if err := o.memory.Append(ctx, tenantID, o.agentID, threadID, spokeEntry); err != nil {
			logger.Warn().Err(err).Msg("failed to append handoff suggestion observation to short-term memory")
		}

		// 2. Concatenate instructions
		delegationMsg := fmt.Sprintf("[Handoff instruction: %s] Original task: %s", handoff.Reason, originalMsg)

		// 3. Update state in Redis
		state.PendingSpokes[handoff.Target] = delegationMsg
		state.Status = model.StatusWaitingSpoke
		if err := o.stateStore.SaveState(ctx, tenantID, threadID, *state); err != nil {
			return fmt.Errorf("failed to save orchestration state for handoff: %w", err)
		}

		// 4. Emit progression update
		progMsg := fmt.Sprintf("Executing handoff from %s to %s...", matchedSpokeKey, handoff.Target)
		if err := o.emitProgressionEvent(ctx, tenantID, threadID, state.OriginalEventID, progMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to publish flow progression event")
		}

		// 5. Context History extraction
		history, err := o.memory.Get(ctx, tenantID, o.agentID, threadID, 15)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to load short-term memory for context history propagation, using empty history")
			history = []model.MemoryEntry{}
		}

		var contextHistory []events.ThreadTurn
		for _, turn := range history {
			contextHistory = append(contextHistory, events.ThreadTurn{
				Role:      turn.Role,
				Content:   turn.Content,
				Timestamp: turn.Timestamp.Format(time.RFC3339),
				Metadata:  turn.Metadata,
			})
		}

		// 6. Publish delegation to target spoke
		taskPayload := events.AgentDelegationPayload{
			ThreadID:        threadID,
			CommunityID:     o.communityID,
			DelegatingAgent: o.agentName,
			TargetAgent:     handoff.Target,
			Message:         delegationMsg,
			ContextHistory:  contextHistory,
		}

		sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
		taskEvent, err := events.NewDomainEvent(
			events.SchemaConversationalAgentDelegation,
			sourceIdentity,
			tenantID,
			taskPayload,
		)
		if err != nil {
			return fmt.Errorf("failed to construct task event for target spoke %s: %w", handoff.Target, err)
		}

		eventData, err := json.Marshal(taskEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal task event: %w", err)
		}

		subject := fmt.Sprintf("ts.community.%s.agent.%s", o.communityID, handoff.Target)
		logger.Info().Str("subject", subject).Str("spoke", handoff.Target).Msg("publishing handoff task to target spoke agent")
		if err := o.publisher.Publish(ctx, subject, eventData); err != nil {
			return fmt.Errorf("failed to publish task to target spoke %s: %w", handoff.Target, err)
		}

		return nil
	}

	// Normal Flow / Fallback when handoff is target-invalid or not requested
	spokeEntry := model.MemoryEntry{
		Role:      "system",
		Content:   fmt.Sprintf("[Observation] Spoke Agent '%s' responded: %s", matchedSpokeKey, payload.Response),
		Timestamp: time.Now().UTC(),
	}
	if err := o.memory.Append(ctx, tenantID, o.agentID, threadID, spokeEntry); err != nil {
		logger.Warn().Err(err).Msg("failed to append spoke response to short-term memory")
	}

	// Emit progression update
	progMsg := fmt.Sprintf("Received response from %s.", matchedSpokeKey)
	if err := o.emitProgressionEvent(ctx, tenantID, threadID, state.OriginalEventID, progMsg); err != nil {
		logger.Warn().Err(err).Msg("failed to publish flow progression event")
	}

	// Fan-in / Join check
	if len(state.PendingSpokes) > 0 {
		logger.Info().Int("pending_count", len(state.PendingSpokes)).Msg("partial spoke response received, yielding and waiting for other concurrent spokes")
		if err := o.stateStore.SaveState(ctx, tenantID, threadID, *state); err != nil {
			return fmt.Errorf("failed to save updated state: %w", err)
		}
		return nil
	}

	// All concurrent spokes finished, proceed to next orchestration turn
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

		// Use brain to ensure the final answer is human-readable
		polishedResponse, err := o.ensureHumanReadable(ctx, finalResponse)
		if err == nil {
			finalResponse = polishedResponse
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

	// Since the LLM adapter automatically appends the BrainRequest.Prompt as a user message,
	// and the full conversational history (including all observations) is stored in history,
	// we pass the entire history as History, and use a static coordination instruction as the Prompt.
	// This prevents observations from being formatted as user messages.
	promptForBrain := "Coordinate the next step based on the conversation history and observations. Output a valid JSON response with the action 'delegate' or 'finalize'."
	historyForBrain := history

	// 3. Compile prompt detailing available specialized Spoke agents
	systemPrompt, err := o.compileSystemPrompt(ctx)
	if err != nil {
		return fmt.Errorf("failed to compile orchestrator system prompt: %w", err)
	}

	// 4. Invoke Brain (LLM) for routing turn
	brainReq := model.BrainRequest{
		Prompt:       promptForBrain,
		SystemPrompt: systemPrompt,
		History:      historyForBrain,
	}

	resp, err := o.brain.Generate(ctx, brainReq)
	if err != nil {
		return fmt.Errorf("orchestrator brain generation failed: %w", err)
	}

	// 5. Parse action decision
	var action OrchestrationAction
	cleanedContent := CleanAndExtractJSON(resp.Content)

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

		// Discover card names from registry to normalize casing of spokes
		cards, err := o.discovery.GetCards(ctx)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to load agent cards for normalization, using raw brain output names")
		}

		// Save state with map of pending spokes
		state.Status = model.StatusWaitingSpoke
		state.PendingSpokes = make(map[string]string)
		var spokeNames []string
		var normalizedSpokes []SpokeTask

		for _, task := range action.Spokes {
			officialName := task.Spoke
			if err == nil {
				for _, card := range cards {
					if strings.EqualFold(card.Name, task.Spoke) {
						officialName = card.Name
						break
					}
				}
			}
			state.PendingSpokes[officialName] = task.Message
			spokeNames = append(spokeNames, officialName)
			normalizedSpokes = append(normalizedSpokes, SpokeTask{
				Spoke:   officialName,
				Message: task.Message,
			})
		}

		if err := o.stateStore.SaveState(ctx, tenantID, threadID, *state); err != nil {
			return fmt.Errorf("failed to save waiting state: %w", err)
		}

		// Publish flow progression event: fanning out
		progMsg := fmt.Sprintf("Delegating tasks to: [%s]...", strings.Join(spokeNames, ", "))
		if err := o.emitProgressionEvent(ctx, tenantID, threadID, state.OriginalEventID, progMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to publish flow progression event")
		}

		// Fetch Context History from Hub's STM
		history, err := o.memory.Get(ctx, tenantID, o.agentID, threadID, 15)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to load short-term memory for context history propagation, using empty history")
			history = []model.MemoryEntry{}
		}

		var contextHistory []events.ThreadTurn
		for _, turn := range history {
			contextHistory = append(contextHistory, events.ThreadTurn{
				Role:      turn.Role,
				Content:   turn.Content,
				Timestamp: turn.Timestamp.Format(time.RFC3339),
				Metadata:  turn.Metadata,
			})
		}

		// Publish task events to all targeted Spokes
		sourceIdentity := fmt.Sprintf("agent/%s", o.agentID)
		for _, task := range normalizedSpokes {
			taskPayload := events.AgentDelegationPayload{
				ThreadID:        threadID,
				CommunityID:     o.communityID,
				DelegatingAgent: o.agentName,
				TargetAgent:     task.Spoke,
				Message:         task.Message,
				ContextHistory:  contextHistory,
			}

			taskEvent, err := events.NewDomainEvent(
				events.SchemaConversationalAgentDelegation,
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

		finalResponse := action.Response
		polishedResponse, err := o.ensureHumanReadable(ctx, finalResponse)
		if err == nil {
			finalResponse = polishedResponse
			action.Response = polishedResponse
		}

		// Append final response to memory history
		assistantEntry := model.MemoryEntry{
			Role:      "assistant",
			Content:   finalResponse,
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

	var spokesSb strings.Builder
	for _, card := range cards {
		if card.Name == o.agentName {
			continue
		}
		spokesSb.WriteString(fmt.Sprintf("- Name: %s\n", card.Name))
		if card.Description != "" {
			spokesSb.WriteString(fmt.Sprintf("  Description: %s\n", card.Description))
		}
		if len(card.Skills) > 0 {
			var skills []string
			for _, skill := range card.Skills {
				skills = append(skills, skill.Name)
			}
			spokesSb.WriteString(fmt.Sprintf("  Skills: %s\n", strings.Join(skills, ", ")))
		}
	}
	spokesStr := spokesSb.String()

	// Parse basePrompt to see if it is a JSON PropagatedAgentConfig
	var parsedConfig PropagatedAgentConfig
	var templateStr string
	var descriptionStr string

	if json.Unmarshal([]byte(o.basePrompt), &parsedConfig) == nil {
		templateStr = parsedConfig.Directives
		descriptionStr = parsedConfig.Description
	} else {
		// Fallback for tests/unstructured prompts
		if strings.Contains(o.basePrompt, ".Spokes") {
			templateStr = o.basePrompt
		} else {
			descriptionStr = o.basePrompt
		}
	}

	// If the template string is empty, fallback to the default Hub instructions
	if templateStr == "" {
		templateStr = `{{if .Description}}{{.Description}}{{else}}You are a helpful orchestrator agent.{{end}}

You have access to the following specialized Spoke agents in this community:
{{.Spokes}}

To coordinate the conversation, you must output a valid JSON response specifying your next step.
- To delegate tasks to Spoke subagents concurrently, output:
  {"action": "delegate", "spokes": [{"spoke": "<agent_name>", "message": "<task description>"}, ...] }
- If a specialized Spoke agent asks a clarifying question to the user or indicates that information/details are missing, you must immediately choose the "finalize" action and return that question directly to the user so they can reply.
- If you have completed the user request and want to finalize the response, output:
  {"action": "finalize", "response": "<final response message to the user>"}
- Do not delegate the wait state or try to delegate again if you are waiting for user input.

Dynamic Routing & Delegation Guidelines:
1. Carefully inspect the Name and Description of the available Spoke agents.
2. During the information-gathering phase of a thread (where details are missing or clarifying questions are needed), delegate tasks ONLY to agents whose descriptions indicate they perform inquiry, question-asking, coaching, or detail gathering.
3. Do NOT delegate tasks to synthesis, compiling, or final-answer agents (e.g., agents whose descriptions state they summarize findings or produce final outputs) during the information-gathering phase. Only delegate to them once all details are fully gathered and you are ready to produce the final findings.

Response Synthesis Guidelines:
1. Messages prefixed with "[Observation]" in the conversation history are responses received from Spoke agents — they are NOT user messages.
2. When finalizing, you MUST synthesize and integrate the Spoke observations into a single cohesive, polished response for the user.
3. Do NOT copy-paste or concatenate Spoke responses verbatim. Rewrite and merge them into a well-structured answer.`
	}

	tmpl, err := template.New("hub_system_prompt").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse system prompt template: %w", err)
	}

	data := map[string]interface{}{
		"Description": descriptionStr,
		"Spokes":      spokesStr,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute system prompt template: %w", err)
	}

	return buf.String(), nil
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
		events.SchemaConversationalAgentReasoning,
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

func (o *Orchestrator) ensureHumanReadable(ctx context.Context, text string) (string, error) {
	if o.brain == nil {
		return text, nil
	}

	prompt := fmt.Sprintf("Please review the following response. If it is already a clean, human-readable, and polished message, output it exactly as-is. If it contains raw data, observation logs, or is unstructured, rewrite and polish it to be a clear, cohesive, and human-friendly final answer to the user. Maintain all facts and details. Do not include any explanations, introduction, or conversational filler.\n\nResponse to review:\n%s", text)
	resp, err := o.brain.Generate(ctx, model.BrainRequest{
		Prompt:       prompt,
		SystemPrompt: "You are a polishing assistant. Your task is to output ONLY the final, polished response. Do NOT include any introduction, explanations, meta-commentary, or preamble (such as 'Here is the polished version:', 'I have reviewed the response', or 'It looks like you provided a JSON object'). Simply output the polished response message directly.",
	})
	if err != nil {
		return text, err
	}
	return resp.Content, nil
}
