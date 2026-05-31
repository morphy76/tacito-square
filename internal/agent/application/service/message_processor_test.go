package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBrain is a mock implementation of the Brain outbound port.
type MockBrain struct {
	GenerateFunc func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error)
}

func (m *MockBrain) Generate(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockBrain) GenerateStream(ctx context.Context, request model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error) {
	return nil, nil, nil
}

// MockShortTermMemory is a mock implementation of the ShortTermMemory outbound port.
type MockShortTermMemory struct {
	AppendFunc func(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error
	GetFunc    func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error)
	ClearFunc  func(ctx context.Context, tenantID, agentID, threadID string) error

	AppendCalls []model.MemoryEntry
	GetCalls    []int
}

func (m *MockShortTermMemory) Append(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error {
	m.AppendCalls = append(m.AppendCalls, entry)
	if m.AppendFunc != nil {
		return m.AppendFunc(ctx, tenantID, agentID, threadID, entry)
	}
	return nil
}

func (m *MockShortTermMemory) Get(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
	m.GetCalls = append(m.GetCalls, limit)
	if m.GetFunc != nil {
		return m.GetFunc(ctx, tenantID, agentID, threadID, limit)
	}
	return []model.MemoryEntry{}, nil
}

func (m *MockShortTermMemory) Clear(ctx context.Context, tenantID, agentID, threadID string) error {
	if m.ClearFunc != nil {
		return m.ClearFunc(ctx, tenantID, agentID, threadID)
	}
	return nil
}

func TestMessageProcessorService_ProcessIncomingMessage(t *testing.T) {
	t.Run("should map incoming payload to brain request, maintain history, and return content", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				assert.Equal(t, "Hello, world", request.Prompt)
				require.Len(t, request.History, 1)
				assert.Equal(t, "Hello, world", request.History[0].Content)
				assert.Equal(t, "user", request.History[0].Role)

				return &model.BrainResponse{
					Content: "Reasoning response content",
				}, nil
			},
		}

		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{Role: "user", Content: "Hello, world"},
				}, nil
			},
		}

		var processor inbound.MessageProcessor = service.NewMessageProcessorService(mockBrain, mockMemory)

		res, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello, world")
		assert.NoError(t, err)
		assert.Equal(t, "Reasoning response content", res)

		// Assert double-append calls
		require.Len(t, mockMemory.AppendCalls, 2)
		assert.Equal(t, "user", mockMemory.AppendCalls[0].Role)
		assert.Equal(t, "Hello, world", mockMemory.AppendCalls[0].Content)
		assert.Equal(t, "assistant", mockMemory.AppendCalls[1].Role)
		assert.Equal(t, "Reasoning response content", mockMemory.AppendCalls[1].Content)
	})

	t.Run("should gracefully degrade and process statelessly if memory fails", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				assert.Equal(t, "Hello, world", request.Prompt)
				// History fallback should only contain the current user message turn since Get failed
				require.Len(t, request.History, 1)
				assert.Equal(t, "Hello, world", request.History[0].Content)

				return &model.BrainResponse{
					Content: "Reasoning response content",
				}, nil
			},
		}

		mockMemory := &MockShortTermMemory{
			AppendFunc: func(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error {
				return errors.New("redis connection refused")
			},
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return nil, errors.New("redis get failed")
			},
		}

		processor := service.NewMessageProcessorService(mockBrain, mockMemory)

		// Service must not fail even if memory fails
		res, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello, world")
		assert.NoError(t, err)
		assert.Equal(t, "Reasoning response content", res)
	})

	t.Run("should bubble up brain generation errors", func(t *testing.T) {
		mockErr := errors.New("brain error")
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				return nil, mockErr
			},
		}
		mockMemory := &MockShortTermMemory{}

		processor := service.NewMessageProcessorService(mockBrain, mockMemory)
		_, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello")
		assert.ErrorIs(t, err, mockErr)
	})
}
