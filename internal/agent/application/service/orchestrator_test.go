package service_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

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
		mockMemory := &MockShortTermMemory{}
		mockPublisher := &MockEventPublisherOrchestrator{}

		// Mock Brain returning a delegate action with multiple spokes
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// Assert system prompt contains the spokes info
				assert.Contains(t, request.SystemPrompt, "writer")
				assert.Contains(t, request.SystemPrompt, "translator")

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
		assert.Equal(t, "waiting_spoke", state.Status)
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
		assert.Equal(t, events.SchemaConversationalAgentResponse, progressEvt.SchemaRef)
		var progressPayload events.AgentResponsePayload
		err = json.Unmarshal(progressEvt.Payload, &progressPayload)
		require.NoError(t, err)
		assert.False(t, progressPayload.Finished)
		assert.Contains(t, progressPayload.Response, "Delegating tasks to: [writer, translator]")

		// Verification of tasks sent to Spokes
		var taskEvt1, taskEvt2 events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &taskEvt1)
		require.NoError(t, err)
		err = json.Unmarshal(publishes[2].Data, &taskEvt2)
		require.NoError(t, err)

		assert.Equal(t, events.SchemaConversationalAddUserMessage, taskEvt1.SchemaRef)
		assert.Equal(t, events.SchemaConversationalAddUserMessage, taskEvt2.SchemaRef)

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
			Status:      "waiting_spoke",
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
		assert.Equal(t, "waiting_spoke", state.Status)
		assert.Len(t, state.PendingSpokes, 1)
		assert.Contains(t, state.PendingSpokes, "translator")
		assert.NotContains(t, state.PendingSpokes, "writer")

		// Check memory append call was triggered
		assert.Len(t, mockMemory.AppendCalls, 1)
		assert.Contains(t, mockMemory.AppendCalls[0].Content, "writer")

		// Check progression publish (Finished: false)
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 1)
		var progressEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &progressEvt)
		require.NoError(t, err)
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
			Status:      "waiting_spoke",
			PendingSpokes: map[string]string{
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
				respJSON := `{
					"action": "finalize",
					"response": "Here is the completed translation of the dragon story."
				}`
				return &model.BrainResponse{Content: respJSON}, nil
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
		assert.Equal(t, "Here is the completed translation of the dragon story.", finalPayload.Response)
		assert.Equal(t, "ts.community.comm-1.agent.hub-123.thread.thread-abc.response", publishes[1].Subject)
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
			Status:          "waiting_spoke",
			PendingSpokes:   map[string]string{"translator": "task"},
			OriginalEventID: "event-999",
			LoopCount:       4,
			MaxLoops:        4, // limit met
		}
		err := mockStateStore.SaveState(context.Background(), "tenant-1", "thread-abc", initialState)
		require.NoError(t, err)

		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				t.Fatal("Brain should not be called when loop limit is exceeded")
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

		// Check publishes contains the EndThread event and fallback error message
		publishes := mockPublisher.GetPublishes()
		require.Len(t, publishes, 2)

		// 1. Verify EndThread event sent to spokes
		var endEvt events.DomainEvent
		err = json.Unmarshal(publishes[0].Data, &endEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalEndThread, endEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.all", publishes[0].Subject)

		// 2. Verify Final Response sent to Keeper/BFF
		var finalEvt events.DomainEvent
		err = json.Unmarshal(publishes[1].Data, &finalEvt)
		require.NoError(t, err)
		assert.Equal(t, events.SchemaConversationalAgentResponse, finalEvt.SchemaRef)
		assert.Equal(t, "ts.community.comm-1.agent.hub-123.thread.thread-abc.response", publishes[1].Subject)

		var finalPayload events.AgentResponsePayload
		err = json.Unmarshal(finalEvt.Payload, &finalPayload)
		require.NoError(t, err)
		assert.True(t, finalPayload.Finished)
		assert.Contains(t, finalPayload.Response, "Orchestration limit exceeded")
	})
}
