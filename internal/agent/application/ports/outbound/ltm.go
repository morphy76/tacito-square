package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
)

// LongTermMemory defines the outbound port interface for semantic vector search and persistence.
type LongTermMemory interface {
	// Save stores semantic memory entries under a strictly tenant-isolated scope.
	Save(ctx context.Context, tenantID, agentID string, entries []model.LTMEntry) error

	// Search queries Qdrant for similar memories using the provided vector.
	Search(ctx context.Context, tenantID, agentID string, vector []float32, filter model.LTMFilter, limit int, threshold float32) ([]model.LTMEntry, error)

	// Delete removes memories matching specific filters.
	Delete(ctx context.Context, tenantID, agentID string, filter model.LTMFilter) error
}
