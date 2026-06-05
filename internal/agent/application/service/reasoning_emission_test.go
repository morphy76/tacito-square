package service_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockPublisher struct {
	PublishFunc func(ctx context.Context, subject string, data []byte) error
	Calls       []PublishCall
}

type PublishCall struct {
	Subject string
	Data    []byte
}

func (m *MockPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	m.Calls = append(m.Calls, PublishCall{Subject: subject, Data: data})
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, subject, data)
	}
	return nil
}

func TestReasoningEmission_PublisherIntegration(t *testing.T) {
	t.Run("happy path: executes thought step and publishes intermediate event payload on NATS publisher", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				toolCall := map[string]any{
					"thought": "Deciding to recall memory.",
					"tool_call": map[string]any{
						"name": "recall_memory",
						"arguments": map[string]any{
							"query": "database info",
						},
					},
				}
				data, _ := json.Marshal(toolCall)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		mockPublisher := &MockPublisher{}

		engine := service.NewCognitiveEngine(mockBrain, 5, nil).WithPublisher(mockPublisher)

		ctx := context.Background()
		// Limit loop execution by invoking recall_memory tool execution that returns error/unavailable
		_, err := engine.ExecuteReasoningLoop(ctx, "tenant-1", "agent-1", "thread-1", "Find database pooling size", []model.MemoryEntry{}, "")
		assert.NoError(t, err)

		// Assert that intermediate events are published
		require.NotEmpty(t, mockPublisher.Calls)

		// Assert NATS Subject format
		assert.Equal(t, "ts.tenant.tenant-1.agent.agent-1.thread.thread-1.reasoning", mockPublisher.Calls[0].Subject)

		// Assert NATS Payload contains intermediate thought and tool calls
		var payload model.AgentReasoningStepPayload
		err = json.Unmarshal(mockPublisher.Calls[0].Data, &payload)
		assert.NoError(t, err)

		assert.Equal(t, 1, payload.StepIndex)
		assert.Equal(t, "Deciding to recall memory.", payload.Thought)
		assert.Equal(t, "recall_memory", payload.Action.Tool)
		assert.Equal(t, "database info", payload.Action.Input["query"])
		assert.False(t, payload.Timestamp.IsZero())
	})
}
