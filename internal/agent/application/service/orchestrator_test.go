package service_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for Orchestrator tests

type MockThreadLock struct {
	mu          sync.Mutex
	locks       map[string]bool
	lockCalls   int
	unlockCalls int
}

func (m *MockThreadLock) Lock(ctx context.Context, tenantID, threadID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockCalls++
	key := tenantID + ":" + threadID
	if m.locks == nil {
		m.locks = make(map[string]bool)
	}
	m.locks[key] = true
	return true, nil
}

func (m *MockThreadLock) Unlock(ctx context.Context, tenantID, threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unlockCalls++
	key := tenantID + ":" + threadID
	if m.locks != nil {
		delete(m.locks, key)
	}
	return nil
}

type MockOrchestrationStateStore struct {
	mu     sync.Mutex
	states map[string]model.OrchestrationState
}

func (m *MockOrchestrationStateStore) SaveState(ctx context.Context, tenantID, threadID string, state model.OrchestrationState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.states == nil {
		m.states = make(map[string]model.OrchestrationState)
	}
	m.states[tenantID+":"+threadID] = state
	return nil
}

func (m *MockOrchestrationStateStore) GetState(ctx context.Context, tenantID, threadID string) (*model.OrchestrationState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[tenantID+":"+threadID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MockOrchestrationStateStore) ClearState(ctx context.Context, tenantID, threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.states != nil {
		delete(m.states, tenantID+":"+threadID)
	}
	return nil
}

type MockAgentDiscovery struct {
	cards []*agentcard.AgentCard
}

func (m *MockAgentDiscovery) GetCards(ctx context.Context) ([]*agentcard.AgentCard, error) {
	return m.cards, nil
}

type MockEventPublisherOrchestrator struct {
	mu        sync.Mutex
	publishes []MockPublishCall
}

type MockPublishCall struct {
	Subject string
	Data    []byte
}

func (m *MockEventPublisherOrchestrator) Publish(ctx context.Context, subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishes = append(m.publishes, MockPublishCall{Subject: subject, Data: data})
	return nil
}

func (m *MockEventPublisherOrchestrator) GetPublishes() []MockPublishCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MockPublishCall, len(m.publishes))
	copy(out, m.publishes)
	return out
}

func TestOrchestrator_ProcessUserMessage(t *testing.T) {
	t.Run("first turn: acquire lock, read spokes, run brain, delegate concurrently to two spokes, yield, save state, publish tasks, release lock", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{
			cards: []*agentcard.AgentCard{
				{Name: "writer", Description: "Writes things"},
				{Name: "translator", Description: "Translates things"},
			},
		}
		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{
						Role:    "user",
						Content: "Do writer and translator tasks.",
					},
				}, nil
			},
		}
		mockPublisher := &MockEventPublisherOrchestrator{}

		// Mock Brain returning a delegate action with multiple spokes
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// Assert system prompt contains the spokes info
				assert.Contains(t, request.SystemPrompt, "writer")
				assert.Contains(t, request.SystemPrompt, "translator")
				assert.Contains(t, request.SystemPrompt, "Dynamic Routing & Delegation Guidelines")

				// Return delegation JSON
				respJSON := `{
					"action": "delegate",
					"spokes": [
						{"spoke": "writer", "message": "write about a dragon"},
						{"spoke": "translator", "message": "translate 'hello'"}
					]
				}`
				return &model.BrainResponse{Content: respJSON}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		payload := events.AddUserMessagePayload{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Message:     "Do writer and translator tasks.",
		}

		err := orchestrator.ProcessUserMessage(context.Background(), "tenant-1", "thread-abc", payload, "event-999")
		// The test must fail (RED) initially since implementation is not yet completed
		assert.NoError(t, err)

		// Assertions (should pass once GREEN is implemented)
		assert.Equal(t, 1, mockLock.lockCalls)
		assert.Equal(t, 1, mockLock.unlockCalls)

		// Check saved state
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.StatusWaitingSpoke, state.Status)
		assert.Len(t, state.PendingSpokes, 2)
		assert.Equal(t, "write about a dragon", state.PendingSpokes["writer"])
		assert.Equal(t, "translate 'hello'", state.PendingSpokes["translator"])
		assert.Equal(t, "event-999", state.OriginalEventID)
		assert.Equal(t, 1, state.LoopCount)
		assert.Equal(t, 5, state.MaxLoops) // 2 spokes + 3 = 5

		// Check NATS publishes
		publishes := mockPublisher.GetPublishes()
		// Expect 3 publishes:
		// 1. Flow progression event (Finished: false)
		// 2. Task event to writer
		// 3. Task event to translator
		require.Len(t, publishes, 3)

		// Verification of flow progression event
		var progressEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &progressEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentReasoning, progressEvt.SchemaRef)
		var progressPayload events.AgentResponsePayload
		err = json.Unmarshal(progressEvt.Payload, &progressPayload)
		require.NoError(t, err)
		assert.False(t, progressPayload.Finished)
		assert.Contains(t, progressPayload.Response, "Delegating tasks to: [writer, translator]")

		// Verification of tasks sent to Spokes — must use AgentDelegation schema
		var taskEvt1, taskEvt2 events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &taskEvt1)
		require.NoError(t, err)
		err = json.Unmarshal(publishes[2].Data, &taskEvt2)
		require.NoError(t, err)

		assert.Equal(t, events.SchemaConversationalAgentDelegation, taskEvt1.SchemaRef)
		assert.Equal(t, events.SchemaConversationalAgentDelegation, taskEvt2.SchemaRef)

		var payload1, payload2 events.AgentDelegationPayload
		err = json.Unmarshal(taskEvt1.Payload, &payload1)
		require.NoError(t, err)
		err = json.Unmarshal(taskEvt2.Payload, &payload2)
		require.NoError(t, err)

		assert.Len(t, payload1.ContextHistory, 1)
		assert.Equal(t, "Do writer and translator tasks.", payload1.ContextHistory[0].Content)
		assert.Len(t, payload2.ContextHistory, 1)
		assert.Equal(t, "Do writer and translator tasks.", payload2.ContextHistory[0].Content)

		// Ensure correct NATS subjects were targeted
		subjects := []string{publishes[1].Subject, publishes[2].Subject}
		assert.Contains(t, subjects, "ts.community.comm-1.agent.writer")
		assert.Contains(t, subjects, "ts.community.comm-1.agent.translator")
	})
}

func TestOrchestrator_ProcessSpokeResponse(t *testing.T) {
	t.Run("fan-in step: receive first spoke response, update pending list, yield, do not run brain", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		// Populate active state with 2 pending spokes
		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"writer":     "task detail",
				"translator": "task detail",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				t.Fatal("Brain should not be invoked during partial fan-in yield")
				return nil, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "writer",
			CorrelationEventID: "event-writer-task",
			Response:           "Once upon a time there was a dragon...",
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// Check that thread lock was acquired and released
		assert.Equal(t, 1, mockLock.lockCalls)
		assert.Equal(t, 1, mockLock.unlockCalls)

		// Assert updated state in Redis: only 'translator' remains pending
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.StatusWaitingSpoke, state.Status)
		assert.Len(t, state.PendingSpokes, 1)
		assert.Contains(t, state.PendingSpokes, "translator")
		assert.NotContains(t, state.PendingSpokes, "writer")

		// Check memory append call was triggered with "system" role and [Observation] prefix
		assert.Len(t, mockMemory.AppendCalls, 1)
		assert.Equal(t, "system", mockMemory.AppendCalls[0].Role)
		assert.Contains(t, mockMemory.AppendCalls[0].Content, "[Observation]")
		assert.Contains(t, mockMemory.AppendCalls[0].Content, "writer")

		// Check progression publish (Finished: false)
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 1)
		var progressEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &progressEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentReasoning, progressEvt.SchemaRef)
		var progressPayload events.AgentResponsePayload
		err = json.Unmarshal(progressEvt.Payload, &progressPayload)
		require.NoError(t, err)
		assert.False(t, progressPayload.Finished)
		assert.Contains(t, progressPayload.Response, "Received response from writer")
	})

	t.Run("join and finalize step: receive final spoke response, run brain, finalize, clear state, publish final answer, release lock", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		// Populate active state with only 'translator' pending
		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"translator": "task detail",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		calls := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				calls++
				if calls == 1 {
					respJSON := `{
						"action": "finalize",
						"response": "Here is the completed translation of the dragon story."
					}`
					return &model.BrainResponse{Content: respJSON}, nil
				}
				// Polishing call
				return &model.BrainResponse{Content: "Polished final translation"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "translator",
			CorrelationEventID: "event-translator-task",
			Response:           "Il était une fois un dragon...",
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// Assert updated state in Redis: state cleared/deleted
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		assert.Nil(t, state)

		// Check publishes: 
		// 1. Flow progression event for translator response
		// 2. Final response event (Finished: true)
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 2)

		var finalEvt events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &finalEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentResponse, finalEvt.SchemaRef)

		var finalPayload events.AgentResponsePayload
		err = json.Unmarshal(finalEvt.Payload, &finalPayload)
		require.NoError(t, err)
		assert.True(t, finalPayload.Finished)
		assert.Equal(t, "Polished final translation", finalPayload.Response)
		assert.Equal(t, "ts.community.comm-1.agent.hub-123.thread.thread-abc.response", publishes[1].Subject)
	})

	t.Run("handoff execution: parse handoff suggestion, validate target exists, log observation, update state, publish delegation event with concatenated message and ContextHistory, release lock", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{
			cards: []*agentcard.AgentCard{
				{Name: "translator"},
			},
		}
		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{Role: "user", Content: "Translate hello", Timestamp: time.Now().UTC()},
				}, nil
			},
		}
		mockPublisher := &MockEventPublisherOrchestrator{}

		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"writer": "please write a story",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				t.Fatal("Brain should not be invoked during direct handoff delegation execution")
				return nil, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "writer",
			CorrelationEventID: "event-writer-task",
			Response:           `{"action": "suggest_handoff", "target": "translator", "reason": "The text is not in English."}`,
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// Assert updated state in Redis: state updated to wait for translator
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, model.StatusWaitingSpoke, state.Status)
		assert.Len(t, state.PendingSpokes, 1)
		assert.Contains(t, state.PendingSpokes, "translator")
		assert.Equal(t, "[Handoff instruction: The text is not in English.] Original task: please write a story", state.PendingSpokes["translator"])

		// Check memory append call was triggered with the [Observation] handoff format
		require.Len(t, mockMemory.AppendCalls, 1)
		assert.Equal(t, "system", mockMemory.AppendCalls[0].Role)
		assert.Contains(t, mockMemory.AppendCalls[0].Content, "[Observation] Spoke Agent 'writer' suggested handoff to 'translator' because: The text is not in English.")

		// Verify NATS delegation event publishes:
		// Expecting 2 publishes: 1 flow progression update, 1 delegation task to translator
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 2)

		var progressEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &progressEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentReasoning, progressEvt.SchemaRef)

		var taskEvt events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &taskEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentDelegation, taskEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.translator", publishes[1].Subject)

		var taskPayload events.AgentDelegationPayload
		err = json.Unmarshal(taskEvt.Payload, &taskPayload)
		require.NoError(t, err)
		assert.Equal(t, "translator", taskPayload.TargetAgent)
		assert.Equal(t, "hub-agent", taskPayload.DelegatingAgent)
		assert.Equal(t, "[Handoff instruction: The text is not in English.] Original task: please write a story", taskPayload.Message)
		require.Len(t, taskPayload.ContextHistory, 1)
		assert.Equal(t, "Translate hello", taskPayload.ContextHistory[0].Content)
	})

	t.Run("handoff execution: fallback to normal loop if target validation fails", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{
			cards: []*agentcard.AgentCard{
				{Name: "some-other-spoke"},
			},
		}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"writer": "please write a story",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		brainCalls := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				brainCalls++
				if brainCalls == 1 {
					respJSON := `{
						"action": "finalize",
						"response": "Fallback finalize decision"
					}`
					return &model.BrainResponse{Content: respJSON}, nil
				}
				return &model.BrainResponse{Content: "Polished response"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "writer",
			CorrelationEventID: "event-writer-task",
			Response:           `{"action": "suggest_handoff", "target": "missing-agent", "reason": "Missing target."}`,
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// Assert that brain was invoked (fallback triggered)
		assert.Equal(t, 2, brainCalls) // 1 generate routing decision + 1 polishing call
	})
}

func TestOrchestrator_LoopDetection(t *testing.T) {
	t.Run("infinite loop protection: force-finalize when LoopCount > MaxLoops", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		// Populate state with LoopCount equal to MaxLoops
		initialState := model.OrchestrationState{
			ThreadID:        "thread-abc",
			CommunityID:     "comm-1",
			Status:          model.StatusWaitingSpoke,
			PendingSpokes:   map[string]string{"translator": "task"},
			OriginalEventID: "event-999",
			LoopCount:       4,
			MaxLoops:        4, // limit met
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				return &model.BrainResponse{Content: "Polished fallback response"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "translator",
			CorrelationEventID: "event-task",
			Response:           "Spoke response",
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// Assert state is cleared
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		assert.Nil(t, state)

		// Check publishes contains the progression update and final response event
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 2)

		// Verify progression update
		var progEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &progEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentReasoning, progEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.hub-123.thread.thread-abc.response", publishes[0].Subject)

		// Verify Final Response sent to Keeper/BFF contains the latest spoke response ("Spoke response")
		var finalEvt events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &finalEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentResponse, finalEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.hub-123.thread.thread-abc.response", publishes[1].Subject)

		var finalPayload events.AgentResponsePayload
		err = json.Unmarshal(finalEvt.Payload, &finalPayload)
		require.NoError(t, err)
		assert.True(t, finalPayload.Finished)
		assert.Equal(t, "Polished fallback response", finalPayload.Response)
	})
}

func TestOrchestrator_TemplateCompilation(t *testing.T) {
	mockLock := &MockThreadLock{}
	mockStateStore := &MockOrchestrationStateStore{}
	mockDiscovery := &MockAgentDiscovery{
		cards: []*agentcard.AgentCard{
			{Name: "researcher", Description: "Searches for info"},
			{Name: "coder", Description: "Writes code"},
		},
	}
	mockMemory := &MockShortTermMemory{}
	mockPublisher := &MockEventPublisherOrchestrator{}

	// JSON config resembling production:
	jsonConfig := `{
		"description": "You are a dynamic coordinator agent.",
		"directives": "Custom directives template.\nDescription: {{.Description}}\nSpokes list:\n{{.Spokes}}\nGuidelines:\nCoordinate spokes correctly."
	}`

	var compiledPrompt string
	calls := 0
	mockBrain := &MockBrain{
		GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
			calls++
			if calls == 1 {
				compiledPrompt = request.SystemPrompt
				return &model.BrainResponse{Content: `{"action": "finalize", "response": "done"}`}, nil
			}
			return &model.BrainResponse{Content: "polished done"}, nil
		},
	}

	orchestrator := service.NewOrchestrator(
		"hub-123", "hub-agent", "comm-1",
		mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
		jsonConfig,
	)

	payload := events.AddUserMessagePayload{
		ThreadID:    "thread-abc",
		CommunityID: "comm-1",
		Message:     "Compile template test.",
	}

	err := orchestrator.ProcessUserMessage(context.Background(), "tenant-1", "thread-abc", payload, "event-999")
	assert.NoError(t, err)

	// Verify the template was executed and placeholders resolved:
	assert.Contains(t, compiledPrompt, "Custom directives template.")
	assert.Contains(t, compiledPrompt, "Description: You are a dynamic coordinator agent.")
	assert.Contains(t, compiledPrompt, "Spokes list:")
	assert.Contains(t, compiledPrompt, "- Name: researcher")
	assert.Contains(t, compiledPrompt, "  Description: Searches for info")
	assert.Contains(t, compiledPrompt, "- Name: coder")
	assert.Contains(t, compiledPrompt, "  Description: Writes code")
	assert.Contains(t, compiledPrompt, "Guidelines:\nCoordinate spokes correctly.")
}

func TestOrchestrator_CaseInsensitiveRouting(t *testing.T) {
	t.Run("delegate normalization: brain outputs lowercase, registry is capitalized, pending state and NATS subject use official capitalized name", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{
			cards: []*agentcard.AgentCard{
				{Name: "Translator", Description: "Translates text"},
			},
		}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// Brain outputs lowercase 'translator'
				respJSON := `{
					"action": "delegate",
					"spokes": [{"spoke": "translator", "message": "translate hello"}]
				}`
				return &model.BrainResponse{Content: respJSON}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		payload := events.AddUserMessagePayload{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Message:     "Hello world",
		}

		err := orchestrator.ProcessUserMessage(context.Background(), "tenant-1", "thread-abc", payload, "event-999")
		assert.NoError(t, err)

		// Assert pending spoke state in Redis is normalized to official name "Translator"
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Contains(t, state.PendingSpokes, "Translator")
		assert.NotContains(t, state.PendingSpokes, "translator")

		// Assert published NATS message subject uses normalized name "Translator"
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 2) // 1 reasoning/progression event, 1 delegation event
		
		var delegationEvt events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &delegationEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentDelegation, delegationEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.Translator", publishes[1].Subject)

		var delegationPayload events.AgentDelegationPayload
		err = json.Unmarshal(delegationEvt.Payload, &delegationPayload)
		require.NoError(t, err)
		assert.Equal(t, "Translator", delegationPayload.TargetAgent)
	})

	t.Run("response match: spoke responds with capitalized name, pending is capitalized, matches, clears, finalizes", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"Translator": "task detail",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		calls := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				calls++
				if calls == 1 {
					respJSON := `{"action": "finalize", "response": "done"}`
					return &model.BrainResponse{Content: respJSON}, nil
				}
				return &model.BrainResponse{Content: "polished done"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		// Spoke responds with capitalized name "Translator" (or lowercase "translator")
		spokeResponse := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "translator", // Test case-insensitivity on lookup
			CorrelationEventID: "event-task",
			Response:           "Bonjour",
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", spokeResponse)
		assert.NoError(t, err)

		// State cleared/deleted
		state, err := mockStateStore.GetState(context.Background(), "tenant-1", "thread-abc")
		require.NoError(t, err)
		assert.Nil(t, state)

		// Assert memory entry uses normalized name
		assert.Len(t, mockMemory.AppendCalls, 2) // 1 user/observation entry, 1 assistant entry
		assert.Contains(t, mockMemory.AppendCalls[0].Content, "Spoke Agent 'Translator' responded")
	})
}

func TestOrchestrator_HistorySplitting(t *testing.T) {
	t.Run("user message: history is split correctly", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}
		
		userMessage := "Hello, I need help"
		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{
						Role:    "user",
						Content: userMessage,
					},
				}, nil
			},
		}
		mockPublisher := &MockEventPublisherOrchestrator{}

		var capturedReq model.BrainRequest
		calls := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				calls++
				if calls == 1 {
					capturedReq = request
					respJSON := `{"action": "finalize", "response": "done"}`
					return &model.BrainResponse{Content: respJSON}, nil
				}
				return &model.BrainResponse{Content: "polished done"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		payload := events.AddUserMessagePayload{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Message:     userMessage,
		}

		err := orchestrator.ProcessUserMessage(context.Background(), "tenant-1", "thread-abc", payload, "event-999")
		assert.NoError(t, err)

		// Assert that the latest message is preserved in history and static prompt is passed to Brain
		assert.Equal(t, "Coordinate the next step based on the conversation history and observations. Output a valid JSON response with the action 'delegate' or 'finalize'.", capturedReq.Prompt)
		require.Len(t, capturedReq.History, 1)
		assert.Equal(t, userMessage, capturedReq.History[0].Content)
	})

	t.Run("spoke response: history is split correctly with wrapped observation", func(t *testing.T) {
		mockLock := &MockThreadLock{}
		mockStateStore := &MockOrchestrationStateStore{}
		mockDiscovery := &MockAgentDiscovery{}

		// Populate active state with pending spoke
		initialState := model.OrchestrationState{
			ThreadID:    "thread-abc",
			CommunityID: "comm-1",
			Status:      model.StatusWaitingSpoke,
			PendingSpokes: map[string]string{
				"enquirer": "please ask questions",
			},
			OriginalEventID: "event-999",
			LoopCount:       1,
			MaxLoops:        5,
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		spokeResponse := "Hello Riccardo"
		expectedObservation := "[Observation] Spoke Agent 'enquirer' responded: Hello Riccardo"

		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{
						Role:    "user",
						Content: "Hello, my name is Riccardo",
					},
					{
						Role:    "system",
						Content: expectedObservation,
					},
				}, nil
			},
		}
		mockPublisher := &MockEventPublisherOrchestrator{}

		var capturedReq model.BrainRequest
		calls := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				calls++
				if calls == 1 {
					capturedReq = request
					respJSON := `{"action": "finalize", "response": "done"}`
					return &model.BrainResponse{Content: respJSON}, nil
				}
				return &model.BrainResponse{Content: "polished done"}, nil
			},
		}

		orchestrator := service.NewOrchestrator(
			"hub-123", "hub-agent", "comm-1",
			mockBrain, mockStateStore, mockLock, mockDiscovery, mockMemory, mockPublisher,
			"You are the coordinator.",
		)

		payload := events.AgentResponsePayload{
			ThreadID:           "thread-abc",
			CommunityID:        "comm-1",
			AgentName:          "enquirer",
			CorrelationEventID: "event-task",
			Response:           spokeResponse,
			Finished:           true,
		}

		err = orchestrator.ProcessSpokeResponse(context.Background(), "tenant-1", "thread-abc", payload)
		assert.NoError(t, err)

		// Assert that the full history is passed and the static coordination prompt is used
		assert.Equal(t, "Coordinate the next step based on the conversation history and observations. Output a valid JSON response with the action 'delegate' or 'finalize'.", capturedReq.Prompt)
		require.Len(t, capturedReq.History, 2)
		assert.Equal(t, "Hello, my name is Riccardo", capturedReq.History[0].Content)
		assert.Equal(t, expectedObservation, capturedReq.History[1].Content)
		assert.Equal(t, "system", capturedReq.History[1].Role)
	})
}

