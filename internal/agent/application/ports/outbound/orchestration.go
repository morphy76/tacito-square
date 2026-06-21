package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
)

// OrchestrationStateStore defines the outbound port for saving, retrieving, and clearing orchestration state.
type OrchestrationStateStore interface {
	SaveState(ctx context.Context, tenantID, threadID string, state model.OrchestrationState) error
	GetState(ctx context.Context, tenantID, threadID string) (*model.OrchestrationState, error)
	ClearState(ctx context.Context, tenantID, threadID string) error
}

// ThreadLock defines the outbound port for acquiring and releasing distributed locks on conversation threads.
type ThreadLock interface {
	Lock(ctx context.Context, tenantID, threadID string) (string, bool, error)
	Unlock(ctx context.Context, tenantID, threadID, token string) error
}

// AgentDiscovery defines the outbound port for listing agent cards in the community registry.
// Agent names are unique within a tenant's community, and since subjects include
// the community UUID, name-based routing is globally unique.
type AgentDiscovery interface {
	GetCards(ctx context.Context) ([]*agentcard.AgentCard, error)
}
