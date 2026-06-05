package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/spf13/viper"
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

// MockEmbedder is a mock implementation of the Embedder outbound port.
type MockEmbedder struct {
	CreateEmbeddingFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockEmbedder) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if m.CreateEmbeddingFunc != nil {
		return m.CreateEmbeddingFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbedder) CreateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{}, nil
}

// MockLongTermMemory is a mock implementation of the LongTermMemory outbound port.
type MockLongTermMemory struct {
	mu         sync.Mutex
	SaveFunc   func(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error
	SearchFunc func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error)
	DeleteFunc func(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error

	SaveCalls []model.LTMEntry
}

func (m *MockLongTermMemory) Save(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error {
	m.mu.Lock()
	m.SaveCalls = append(m.SaveCalls, entries...)
	m.mu.Unlock()
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, tenantID, agentID, entries)
	}
	return nil
}

func (m *MockLongTermMemory) GetSaveCalls() []model.LTMEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]model.LTMEntry, len(m.SaveCalls))
	copy(calls, m.SaveCalls)
	return calls
}

func (m *MockLongTermMemory) Search(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, tenantID, agentID, vector, filter, limit, threshold)
	}
	return []model.LTMEntry{}, nil
}

func (m *MockLongTermMemory) Delete(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, tenantID, agentID, filter)
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

		mockLTM := &MockLongTermMemory{}
		mockEmbed := &MockEmbedder{}

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, nil).WithLTM(mockEmbed, mockLTM)

		var processor inbound.MessageProcessor = service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 10, "", nil)

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
				// When Get fails, history falls back to empty: the current user message is
				// already carried by req.Prompt and must not be duplicated in history.
				assert.Empty(t, request.History)

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

		mockLTM := &MockLongTermMemory{}
		mockEmbed := &MockEmbedder{}

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, nil).WithLTM(mockEmbed, mockLTM)

		processor := service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 10, "", nil)

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
		mockLTM := &MockLongTermMemory{}
		mockEmbed := &MockEmbedder{}

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, nil).WithLTM(mockEmbed, mockLTM)

		processor := service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 10, "", nil)
		_, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello")
		assert.ErrorIs(t, err, mockErr)
	})

	t.Run("should query LTM, inject retrieved matches, and consolidate on eviction", func(t *testing.T) {
		stepCount := 0
		// Mock brain that outputs active recall memory tool request, then final answer
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				// If it's a summarization call (the background worker runs async summarization)
				if strings.Contains(request.Prompt, "summarize") || strings.Contains(request.Prompt, "compress") {
					return &model.BrainResponse{
						Content: "Summary of evicted turns",
					}, nil
				}

				stepCount++
				if stepCount == 1 {
					toolCall := map[string]any{
						"thought": "Deciding to recall memories.",
						"tool_call": map[string]any{
							"name": "recall_memory",
							"arguments": map[string]any{
								"query": "Hello",
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				// Verify that tool observation was successfully injected into history turns
				assert.True(t, len(request.History) > 0)
				lastTurn := request.History[len(request.History)-1]
				assert.Equal(t, "tool", lastTurn.Role)
				assert.Contains(t, lastTurn.Content, "cognitive context matched")

				finalAnswer := map[string]any{
					"final_answer": "LLM Response",
				}
				data, _ := json.Marshal(finalAnswer)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		// Mock stm with 4 turns to trigger consolidation (limit is set to 3)
		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{Role: "user", Content: "Turn 1", Timestamp: time.Now()},
					{Role: "assistant", Content: "Resp 1", Timestamp: time.Now()},
					{Role: "user", Content: "Turn 2", Timestamp: time.Now()},
					{Role: "assistant", Content: "Resp 2", Timestamp: time.Now()},
				}, nil
			},
		}

		mockLTM := &MockLongTermMemory{
			SearchFunc: func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
				return []model.LTMEntry{
					{ID: "m1", Content: "cognitive context matched", Score: 0.9},
				}, nil
			},
		}

		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				return []float32{0.1, 0.2, 0.3}, nil
			},
		}

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, nil).WithLTM(mockEmbed, mockLTM)
		processor := service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 3, "", nil)

		res, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello")
		assert.NoError(t, err)
		assert.Equal(t, "LLM Response", res)

		// Wait briefly for the async consolidation goroutine to complete
		time.Sleep(100 * time.Millisecond)

		// Verify that LTM Save was called with the summarized eviction
		saveCalls := mockLTM.GetSaveCalls()
		require.NotEmpty(t, saveCalls)
		assert.Equal(t, model.EntryTypeConversation, saveCalls[0].Type)
		assert.Contains(t, saveCalls[0].Content, "Summary of evicted turns")
	})

	t.Run("should gracefully degrade if LTM or Embedder fails", func(t *testing.T) {
		stepCount := 0
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				stepCount++
				if stepCount == 1 {
					toolCall := map[string]any{
						"tool_call": map[string]any{
							"name": "recall_memory",
							"arguments": map[string]any{
								"query": "Hello",
							},
						},
					}
					data, _ := json.Marshal(toolCall)
					return &model.BrainResponse{Content: string(data)}, nil
				}

				// The history contains the tool error observation from graceful degradation
				assert.True(t, len(request.History) > 0)
				lastTurn := request.History[len(request.History)-1]
				assert.Equal(t, "tool", lastTurn.Role)
				assert.Contains(t, lastTurn.Content, "Memory store temporarily unavailable")

				finalAnswer := map[string]any{
					"final_answer": "Response without LTM",
				}
				data, _ := json.Marshal(finalAnswer)
				return &model.BrainResponse{Content: string(data)}, nil
			},
		}

		mockMemory := &MockShortTermMemory{}
		mockLTM := &MockLongTermMemory{
			SearchFunc: func(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error) {
				return nil, errors.New("qdrant timeout")
			},
		}
		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				return nil, errors.New("embeddings api outage")
			},
		}

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, nil).WithLTM(mockEmbed, mockLTM)
		processor := service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 10, "", nil)

		res, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello")
		assert.NoError(t, err)
		assert.Equal(t, "Response without LTM", res)
	})

	t.Run("should bypass memory consolidation on eviction if bypass.ltm is set to true", func(t *testing.T) {
		mockBrain := &MockBrain{
			GenerateFunc: func(ctx context.Context, request model.BrainRequest) (*model.BrainResponse, error) {
				return &model.BrainResponse{Content: "Reasoning response content"}, nil
			},
		}

		mockMemory := &MockShortTermMemory{
			GetFunc: func(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
				return []model.MemoryEntry{
					{Role: "user", Content: "Turn 1", Timestamp: time.Now()},
					{Role: "assistant", Content: "Resp 1", Timestamp: time.Now()},
					{Role: "user", Content: "Turn 2", Timestamp: time.Now()},
					{Role: "assistant", Content: "Resp 2", Timestamp: time.Now()},
				}, nil
			},
		}

		mockLTM := &MockLongTermMemory{}
		mockEmbed := &MockEmbedder{
			CreateEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
				t.Error("Embedder should not be called when bypass.ltm is true")
				return nil, errors.New("should not be called")
			},
		}

		// Configure viper with bypass.ltm set to true
		cfg := viper.New()
		cfg.Set("bypass.ltm", true)

		cogEngine := service.NewCognitiveEngine(mockBrain, 5, cfg).WithLTM(mockEmbed, mockLTM)
		processor := service.NewMessageProcessorService(mockBrain, mockMemory, mockLTM, mockEmbed, cogEngine, 3, "", cfg)

		res, err := processor.ProcessIncomingMessage(context.Background(), "tenant-1", "agent-1", "thread-123", "Hello")
		assert.NoError(t, err)
		assert.Equal(t, "Reasoning response content", res)

		// Wait briefly to make sure async goroutine would have run and finished if it wasn't bypassed
		time.Sleep(50 * time.Millisecond)

		// Assert no calls to Save were made
		assert.Empty(t, mockLTM.GetSaveCalls())
	})
}
