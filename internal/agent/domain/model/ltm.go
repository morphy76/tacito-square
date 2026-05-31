package model

import (
	"errors"
	"time"
)

// LTMEntryType represents the class of semantic content stored in Long-Term Memory.
type LTMEntryType string

const (
	EntryTypeConversation LTMEntryType = "conversation" // Evicted STM turns, summaries of past threads
	EntryTypeDocument     LTMEntryType = "document"     // Chunked loaded documents (PDFs, Markdown, text)
	EntryTypeFact         LTMEntryType = "fact"         // Extracted declarations (user preferences, system settings)
	EntryTypeProcedural   LTMEntryType = "procedural"   // Successful plans, recipes, tool-sequence execution traces
)

// LTMEntry represents a single semantic memory stored as a dense vector with rich metadata.
type LTMEntry struct {
	ID        string            `json:"id"`                  // Unique UUID
	Content   string            `json:"content"`             // Raw text represented by the embedding
	Embedding []float32         `json:"embedding,omitempty"` // High-dimensional dense vector
	Type      LTMEntryType      `json:"type"`                // Type classification of the memory
	Source    string            `json:"source"`              // e.g. "eviction_consolidator", "pdf_uploader", "thought_loop"
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`  // Extra metadata (e.g. document name, page number, user ID)
	Score     float32           `json:"score,omitempty"`     // Match relevance score (populated on search results)
}

// LTMFilter represents runtime constraints for querying long-term memories.
type LTMFilter struct {
	Types       []LTMEntryType `json:"types,omitempty"`        // Filter by specific memory types
	ThreadID    string         `json:"thread_id,omitempty"`    // Optional thread-specific scoping
	CommunityID string         `json:"community_id,omitempty"` // For community-wide sharing
	Visibility  string         `json:"visibility,omitempty"`   // "private", "community", or "tenant"
}

// Validate asserts standard long-term memory constraints.
func (e *LTMEntry) Validate() error {
	if e.ID == "" {
		return errors.New("memory id must not be empty")
	}
	if e.Content == "" {
		return errors.New("memory content must not be empty")
	}
	if len(e.Embedding) == 0 {
		return errors.New("memory embedding vector must not be empty")
	}
	switch e.Type {
	case EntryTypeConversation, EntryTypeDocument, EntryTypeFact, EntryTypeProcedural:
		// valid
	default:
		return errors.New("invalid long-term memory type: " + string(e.Type))
	}
	return nil
}
