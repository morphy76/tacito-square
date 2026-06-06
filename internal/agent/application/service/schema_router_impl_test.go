package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockMessageProcessor struct {
	ProcessIncomingMessageFunc func(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error)
	Calls                      []string
}

func (m *MockMessageProcessor) ProcessIncomingMessage(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
	m.Calls = append(m.Calls, payload)
	if m.ProcessIncomingMessageFunc != nil {
		return m.ProcessIncomingMessageFunc(ctx, tenantID, agentID, threadID, payload)
	}
	return "", nil
}

type MockEventPublisher struct {
	PublishFunc func(ctx context.Context, subject string, data []byte) error
	Calls       []struct {
		Subject string
		Data    []byte
	}
}

func (m *MockEventPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	m.Calls = append(m.Calls, struct {
		Subject string
		Data    []byte
	}{Subject: subject, Data: data})
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, subject, data)
	}
	return nil
}

func TestSchemaRouter_StartThread(t *testing.T) {
	mockMemory := &MockShortTermMemory{
		ClearFunc: func(ctx context.Context, tenantID, agentID, threadID string) error {
			assert.Equal(t, "tenant-1", tenantID)
			assert.Equal(t, "agent-123", agentID)
			assert.Equal(t, "thread-abc", threadID)
			return nil
		},
	}

	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		nil,
		mockMemory,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	payload := events.StartThreadPayload{
		ThreadID:    "thread-abc",
		CommunityID: "community-456",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-1",
		SchemaRef:  events.SchemaConversationalStartThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	err = router.RouteEvent(context.Background(), evt)
	assert.NoError(t, err)
}

func TestSchemaRouter_AddUserMessage_Success(t *testing.T) {
	mockProcessor := &MockMessageProcessor{
		ProcessIncomingMessageFunc: func(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
			assert.Equal(t, "Hello, agent", payload)
			return "LLM Response", nil
		},
	}
	mockPublisher := &MockEventPublisher{}

	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		mockProcessor,
		&MockShortTermMemory{},
		nil,
		nil,
		nil,
		mockPublisher,
		nil,
	)

	payload := events.AddUserMessagePayload{
		ThreadID:    "thread-abc",
		CommunityID: "community-456",
		Message:     "Hello, agent",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-1",
		SchemaRef:  events.SchemaConversationalAddUserMessage,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	err = router.RouteEvent(context.Background(), evt)
	assert.NoError(t, err)

	require.Len(t, mockProcessor.Calls, 1)
	require.Len(t, mockPublisher.Calls, 1)

	// Verify NATS subject and payload structure
	pubCall := mockPublisher.Calls[0]
	assert.Equal(t, "ts.community.community-456.agent.test-agent.response", pubCall.Subject)

	var publishedEvent events.DomainEvent
	err = json.Unmarshal(pubCall.Data, &publishedEvent)
	require.NoError(t, err)

	assert.Equal(t, events.SchemaConversationalAgentResponse, publishedEvent.SchemaRef)
	assert.Equal(t, "agent/agent-123", publishedEvent.Source)
	assert.Equal(t, "tenant-1", publishedEvent.TenantID)

	var respPayload events.AgentResponsePayload
	err = json.Unmarshal(publishedEvent.Payload, &respPayload)
	require.NoError(t, err)

	assert.Equal(t, "thread-abc", respPayload.ThreadID)
	assert.Equal(t, "community-456", respPayload.CommunityID)
	assert.Equal(t, "test-agent", respPayload.AgentName)
	assert.Equal(t, "evt-1", respPayload.CorrelationEventID)
	assert.Equal(t, "LLM Response", respPayload.Response)
	assert.True(t, respPayload.Finished)
}

func TestSchemaRouter_AddUserMessage_LLMFailure_Rollback(t *testing.T) {
	mockProcessor := &MockMessageProcessor{
		ProcessIncomingMessageFunc: func(ctx context.Context, tenantID, agentID, threadID string, payload string) (string, error) {
			return "", errors.New("LLM timeout error")
		},
	}
	mockMemory := &MockShortTermMemory{
		RollbackLastFunc: func(ctx context.Context, tenantID, agentID, threadID string) error {
			assert.Equal(t, "tenant-1", tenantID)
			assert.Equal(t, "agent-123", agentID)
			assert.Equal(t, "thread-abc", threadID)
			return nil
		},
	}
	mockPublisher := &MockEventPublisher{}

	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		mockProcessor,
		mockMemory,
		nil,
		nil,
		nil,
		mockPublisher,
		nil,
	)

	payload := events.AddUserMessagePayload{
		ThreadID:    "thread-abc",
		CommunityID: "community-456",
		Message:     "Hello, agent",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-1",
		SchemaRef:  events.SchemaConversationalAddUserMessage,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	err = router.RouteEvent(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM timeout error")

	require.Len(t, mockMemory.RollbackLastCalls, 1)
	assert.Empty(t, mockPublisher.Calls)
}

func TestSchemaRouter_EndThread_LTM(t *testing.T) {
	mockMemory := &MockShortTermMemory{
		GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
			return []model.MemoryEntry{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
			}, nil
		},
		ClearFunc: func(ctx context.Context, tenantID, agentID, threadID string) error {
			return nil
		},
	}

	mockBrain := &MockBrain{
		GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
			assert.Contains(t, request.Prompt, "Hello")
			assert.Contains(t, request.Prompt, "Hi there")
			return &model.BrainResponse{Content: "Summary of the thread"}, nil
		},
	}

	mockEmbedder := &MockEmbedder{
		CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
			assert.Equal(t, "Summary of the thread", text)
			return []float32{0.5, 0.6}, nil
		},
	}

	mockLTM := &MockLongTermMemory{
		SaveFunc: func(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error {
			assert.Equal(t, "tenant-1", tenantID)
			assert.Equal(t, "agent-123", agentID)
			require.Len(t, entries, 1)
			assert.Equal(t, "Summary of the thread", entries[0].Content)
			assert.Equal(t, []float32{0.5, 0.6}, entries[0].Embedding)
			return nil
		},
	}

	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		nil,
		mockMemory,
		mockLTM,
		mockEmbedder,
		mockBrain,
		nil,
		nil,
	)

	payload := events.EndThreadPayload{
		ThreadID:    "thread-abc",
		CommunityID: "community-456",
		Reason:      "finished",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-1",
		SchemaRef:  events.SchemaConversationalEndThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	err = router.RouteEvent(context.Background(), evt)
	assert.NoError(t, err)
}

func TestSchemaRouter_EndThread_EmitsHistoryEvent(t *testing.T) {
	now := time.Now().UTC()
	mockMemory := &MockShortTermMemory{
		GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
			return []model.MemoryEntry{
				{Role: "user", Content: "Hello", Timestamp: now},
				{Role: "assistant", Content: "Hi there", Timestamp: now},
			}, nil
		},
		ClearFunc: func(ctx context.Context, tenantID, agentID, threadID string) error {
			return nil
		},
	}

	mockPublisher := &MockEventPublisher{}

	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		nil,
		mockMemory,
		nil,
		nil,
		nil,
		mockPublisher,
		nil,
	)

	payload := events.EndThreadPayload{
		ThreadID:    "thread-abc",
		CommunityID: "community-456",
		Reason:      "finished",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	evt := events.DomainEvent{
		EventID:    "evt-1",
		SchemaRef:  events.SchemaConversationalEndThread,
		Source:     "keeper",
		TenantID:   "tenant-1",
		OccurredAt: time.Now().Format(time.RFC3339Nano),
		Payload:    payloadBytes,
	}

	err = router.RouteEvent(context.Background(), evt)
	assert.NoError(t, err)

	require.Len(t, mockPublisher.Calls, 1)
	pubCall := mockPublisher.Calls[0]
	assert.Equal(t, "ts.community.community-456.agent.test-agent.history", pubCall.Subject)

	var publishedEvent events.DomainEvent
	err = json.Unmarshal(pubCall.Data, &publishedEvent)
	require.NoError(t, err)

	assert.Equal(t, events.SchemaConversationalThreadHistory, publishedEvent.SchemaRef)
	assert.Equal(t, "agent/agent-123", publishedEvent.Source)
	assert.Equal(t, "tenant-1", publishedEvent.TenantID)

	var histPayload events.ThreadHistoryPayload
	err = json.Unmarshal(publishedEvent.Payload, &histPayload)
	require.NoError(t, err)

	assert.Equal(t, "thread-abc", histPayload.ThreadID)
	assert.Equal(t, "community-456", histPayload.CommunityID)
	require.Len(t, histPayload.History, 2)
	assert.Equal(t, "user", histPayload.History[0].Role)
	assert.Equal(t, "Hello", histPayload.History[0].Content)
	assert.Equal(t, now.Format(time.RFC3339), histPayload.History[0].Timestamp)
	assert.Equal(t, "assistant", histPayload.History[1].Role)
	assert.Equal(t, "Hi there", histPayload.History[1].Content)
	assert.Equal(t, now.Format(time.RFC3339), histPayload.History[1].Timestamp)
}

func TestSchemaRouter_UnknownSchemaRef(t *testing.T) {
	router := service.NewSchemaRouterImpl(
		"agent-123",
		"test-agent",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	evt := events.DomainEvent{
		EventID:   "evt-1",
		SchemaRef: "urn:tacito:schema:unregistered",
		Source:    "keeper",
		TenantID:  "tenant-1",
		Payload:   []byte(`{}`),
	}

	err := router.RouteEvent(context.Background(), evt)
	assert.NoError(t, err)
}
