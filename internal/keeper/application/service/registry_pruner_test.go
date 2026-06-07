package service_test

import (
	"context"
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
func (m *mockPrunerAgentRepository) UpdateStatus(ctx context.Context, agentID uuid.UUID, status model.AgentStatus) error {
	return nil
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

type mockPrunerCache struct {
	invalidated []string
}

func (m *mockPrunerCache) Get(ctx context.Context, key string, dest interface{}) error { return nil }
func (m *mockPrunerCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}
func (m *mockPrunerCache) Invalidate(ctx context.Context, key string) error {
	m.invalidated = append(m.invalidated, key)
	return nil
}

type mockPrunerEventPublisher struct {
	published []events.DomainEvent
	subjects  []string
}

func (m *mockPrunerEventPublisher) Publish(ctx context.Context, subject string, event events.DomainEvent) error {
	m.published = append(m.published, event)
	m.subjects = append(m.subjects, subject)
	return nil
}

func TestRegistryPruner_Pruning(t *testing.T) {
	agentRepo := &mockPrunerAgentRepository{
		pruned: []agentcard.AgentCommunityRef{
			{
				AgentID:     "agent-1",
				CommunityID: "comm-1",
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
	assert.Contains(t, cacheAdapter.invalidated, "communities:comm-1:agents:agent-1")

	// Assert pruner published offline status NATS event
	assert.Contains(t, publisher.subjects, "ts.community.comm-1.agent.agent-1.status")
	assert.GreaterOrEqual(t, len(publisher.published), 1)
	assert.Contains(t, string(publisher.published[0].Payload), "offline")
}
