package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type mockPrunerAgentRepository struct {
	pruned []agentcard.AgentCommunityRef
	err    error
}

func (m *mockPrunerAgentRepository) Create(ctx context.Context, agent *model.Agent) error { return nil }
func (m *mockPrunerAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	return nil, nil
}
func (m *mockPrunerAgentRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	return nil, nil
}
func (m *mockPrunerAgentRepository) List(ctx context.Context) ([]*model.Agent, error) { return nil, nil }
func (m *mockPrunerAgentRepository) Update(ctx context.Context, agent *model.Agent) error { return nil }
func (m *mockPrunerAgentRepository) Delete(ctx context.Context, id uuid.UUID) error       { return nil }
func (m *mockPrunerAgentRepository) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}
func (m *mockPrunerAgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}
func (m *mockPrunerAgentRepository) UpdateStatus(ctx context.Context, agentID uuid.UUID, status model.AgentStatus) (bool, error) {
	return false, nil
}
func (m *mockPrunerAgentRepository) UpsertRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID, card *agentcard.AgentCard) error {
	return nil
}
func (m *mockPrunerAgentRepository) GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*agentcard.AgentCard, time.Time, error) {
	return nil, time.Time{}, nil
}
func (m *mockPrunerAgentRepository) GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*agentcard.AgentCard, time.Time, error) {
	return nil, time.Time{}, nil
}
func (m *mockPrunerAgentRepository) PruneStaleRegistrations(ctx context.Context, threshold time.Duration) ([]agentcard.AgentCommunityRef, error) {
	return m.pruned, m.err
}
func (m *mockPrunerAgentRepository) DeleteRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	return nil
}

type mockPrunerCache struct {
	mu          sync.Mutex
	invalidated []string
}

func (m *mockPrunerCache) Get(ctx context.Context, key string, dest interface{}) error { return nil }
func (m *mockPrunerCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}
func (m *mockPrunerCache) Invalidate(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidated = append(m.invalidated, key)
	return nil
}
func (m *mockPrunerCache) GetInvalidated() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.invalidated))
	copy(out, m.invalidated)
	return out
}

type mockPrunerEventPublisher struct {
	mu        sync.Mutex
	published []events.DomainEvent
	subjects  []string
}

func (m *mockPrunerEventPublisher) Publish(ctx context.Context, subject string, event events.DomainEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
	m.subjects = append(m.subjects, subject)
	return nil
}
func (m *mockPrunerEventPublisher) GetPublished() []events.DomainEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]events.DomainEvent, len(m.published))
	copy(out, m.published)
	return out
}
func (m *mockPrunerEventPublisher) GetSubjects() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.subjects))
	copy(out, m.subjects)
	return out
}

func TestRegistryPruner_Pruning(t *testing.T) {
	agentRepo := &mockPrunerAgentRepository{
		pruned: []agentcard.AgentCommunityRef{
			{
				AgentID:     "agent-1",
				CommunityID: "comm-1",
				TenantID:    "test-tenant",
			},
		},
	}
	cacheAdapter := &mockPrunerCache{}
	publisher := &mockPrunerEventPublisher{}
	logger := zerolog.New(nil)

	pruner := service.NewRegistryPruner(agentRepo, cacheAdapter, publisher, logger)
	pruner.SetInterval(10 * time.Millisecond)

	ctx := context.Background()
	err := pruner.Start(ctx)
	assert.NoError(t, err)

	// Allow loop to tick
	time.Sleep(25 * time.Millisecond)

	err = pruner.Stop()
	assert.NoError(t, err)

	// Assert pruner evicted stale registration from cache
	assert.Contains(t, cacheAdapter.GetInvalidated(), "communities:comm-1:agents:agent-1")

	// Assert pruner published offline status NATS event
	assert.Contains(t, publisher.GetSubjects(), "ts.community.comm-1.agent.agent-1.status")
	published := publisher.GetPublished()
	assert.GreaterOrEqual(t, len(published), 1)
	assert.Contains(t, string(published[0].Payload), "offline")
}
