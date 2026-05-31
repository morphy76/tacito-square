package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
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

func TestMessageProcessorService_ProcessIncomingMessage(t *testing.T) {
	t.Run("should map incoming payload to brain request and return content", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				assert.Equal(t, "Hello, world", request.Prompt)
				return &model.BrainResponse{
					Content: "Reasoning response content",
				}, nil
			},
		}

		// Since service package and MessageProcessorService do not exist, this will fail compilation.
		var processor inbound.MessageProcessor = service.NewMessageProcessorService(mockBrain)

		res, err := processor.ProcessIncomingMessage(context.Background(), "Hello, world")
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

		processor := service.NewMessageProcessorService(mockBrain)
		_, err := processor.ProcessIncomingMessage(context.Background(), "Hello")
		assert.ErrorIs(t, err, mockErr)
	})
}
