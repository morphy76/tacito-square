package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
)

// ShortTermMemory represents the outbound port interface governing short-term conversational and plan history access.
type ShortTermMemory interface {
	// Append appends a new memory entry to the specified thread.
	Append(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error

	// Get retrieves the sliding window of conversational history up to a limit.
	Get(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error)

	// Clear deletes all entries associated with the specified thread.
	Clear(ctx context.Context, tenantID, agentID, threadID string) error
}
