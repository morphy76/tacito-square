package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLTMEntryValidation(t *testing.T) {
	t.Run("valid ltm entry", func(t *testing.T) {
		entry := LTMEntry{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Content:   "cognitive semantic context",
			Embedding: []float32{0.1, 0.2, 0.3},
			Type:      EntryTypeConversation,
			Source:    "eviction_consolidator",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing id", func(t *testing.T) {
		entry := LTMEntry{
			Content:   "cognitive semantic context",
			Embedding: []float32{0.1, 0.2, 0.3},
			Type:      EntryTypeConversation,
			Source:    "eviction_consolidator",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory id must not be empty")
	})

	t.Run("missing content", func(t *testing.T) {
		entry := LTMEntry{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Embedding: []float32{0.1, 0.2, 0.3},
			Type:      EntryTypeConversation,
			Source:    "eviction_consolidator",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory content must not be empty")
	})

	t.Run("missing embedding", func(t *testing.T) {
		entry := LTMEntry{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Content:   "cognitive semantic context",
			Type:      EntryTypeConversation,
			Source:    "eviction_consolidator",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory embedding vector must not be empty")
	})

	t.Run("invalid ltm type", func(t *testing.T) {
		entry := LTMEntry{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Content:   "cognitive semantic context",
			Embedding: []float32{0.1, 0.2, 0.3},
			Type:      LTMEntryType("invalid_type"),
			Source:    "eviction_consolidator",
			Timestamp: time.Now(),
		}
		err := entry.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid long-term memory type")
	})
}

func TestLTMEntryJSONSerialization(t *testing.T) {
	entry := LTMEntry{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		Content:   "cognitive semantic context",
		Embedding: []float32{0.1, 0.2, 0.3},
		Type:      EntryTypeDocument,
		Source:    "pdf_uploader",
		Timestamp: time.Date(2026, 5, 31, 15, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"doc_name": "manual.pdf"},
	}

	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	var decoded LTMEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, entry.Content, decoded.Content)
	assert.Equal(t, entry.Embedding, decoded.Embedding)
	assert.Equal(t, entry.Type, decoded.Type)
	assert.Equal(t, entry.Source, decoded.Source)
	assert.True(t, entry.Timestamp.Equal(decoded.Timestamp))
	assert.Equal(t, entry.Metadata["doc_name"], decoded.Metadata["doc_name"])
}
