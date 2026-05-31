package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
)

// MockLongTermMemory is a mock implementation of the LongTermMemory outbound port.
type MockLongTermMemory struct {
	SaveFunc   func(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error
	SearchFunc func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error)
	DeleteFunc func(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error
}

func (m *MockLongTermMemory) Save(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, tenantID, agentID, entries)
	}
	return nil
}

func (m *MockLongTermMemory) Search(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, tenantID, agentID, vector, filter, limit, threshold)
	}
	return nil, nil
}

func (m *MockLongTermMemory) Delete(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, tenantID, agentID, filter)
	}
	return nil
}

// MockEmbedder is a mock implementation of the Embedder outbound port.
type MockEmbedder struct {
	CreateEmbeddingFunc      func(ctx context.Context, text string) ([]float32, error)
	CreateEmbeddingsBatchFunc func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.CreateEmbeddingFunc != nil {
		return m.CreateEmbeddingFunc(ctx, text)
	}
	return nil, nil
}

func (m *MockEmbedder) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if m.CreateEmbeddingsBatchFunc != nil {
		return m.CreateEmbeddingsBatchFunc(ctx, texts)
	}
	return nil, nil
}

func TestLTMInterfaces_Compliance(t *testing.T) {
	var _ outbound.LongTermMemory = (*MockLongTermMemory)(nil)
	var _ outbound.Embedder = (*MockEmbedder)(nil)

	t.Run("mock ltm search maps parameters correctly", func(t *testing.T) {
		called := false
		mock := &MockLongTermMemory{
			SearchFunc: func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
				called = true
				assert.Equal(t, "T1", tenantID)
				assert.Equal(t, "A1", agentID)
				assert.Equal(t, []float32{0.1, 0.2}, vector)
				assert.Equal(t, "Th1", filter.ThreadID)
				assert.Equal(t, 5, limit)
				assert.Equal(t, float32(0.8), threshold)
				return []model.LTMEntry{
					{ID: "m1", Content: "recalled"},
				}, nil
			},
		}

		res, err := mock.Search(context.Background(), "T1", "A1", []float32{0.1, 0.2}, model.LTMFilter{ThreadID: "Th1"}, 5, 0.8)
		assert.NoError(t, err)
		assert.True(t, called)
		assert.Len(t, res, 1)
		assert.Equal(t, "m1", res[0].ID)
		assert.Equal(t, "recalled", res[0].Content)
	})

	t.Run("mock embedder maps parameters correctly", func(t *testing.T) {
		called := false
		mock := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				called = true
				assert.Equal(t, "test text", text)
				return []float32{0.9, 0.8}, nil
			},
		}

		res, err := mock.CreateEmbedding(context.Background(), "test text")
		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, []float32{0.9, 0.8}, res)
	})
}
