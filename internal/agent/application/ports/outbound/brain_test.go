package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

// MockBrain is a mock implementation of the Brain outbound port.
type MockBrain struct {
	GenerateFunc       func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error)
	GenerateStreamFunc func(ctx context.Context, request model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error)
}

func (m *MockBrain) Generate(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockBrain) GenerateStream(ctx context.Context, request model.BrainRequest) (<-chan model.BrainStreamChunk, <-chan error, error) {
	if m.GenerateStreamFunc != nil {
		return m.GenerateStreamFunc(ctx, request)
	}
	return nil, nil, nil
}

func TestBrainPort_InterfaceCompliance(t *testing.T) {
	var _ outbound.Brain = (*MockBrain)(nil)

	t.Run("generate call maps parameters correctly", func(t *testing.T) {
		called := false
		mock := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				called = true
				assert.Equal(t, "test prompt", request.Prompt)
				return &model.BrainResponse{
					Content: "mocked answer",
				}, nil
			},
		}

		resp, err := mock.Generate(context.Background(), model.BrainRequest{Prompt: "test prompt"})
		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, "mocked answer", resp.Content)
	})
}
