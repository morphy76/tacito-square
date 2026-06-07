//go:build integration

package nats_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	keepernats "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/postgres"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/adapters/outbound/cache"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRedis struct {
	store map[string][]byte
}

func newMockRedis() *mockRedis {
	return &mockRedis{store: make(map[string][]byte)}
}

func (m *mockRedis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.store[key] = value
	return nil
}

func (m *mockRedis) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.store[key]
	if !ok {
		return nil, sharedports.ErrCacheMiss
	}
	return val, nil
}

func (m *mockRedis) Del(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}

func startTestNatsServer(t *testing.T) (*server.Server, *nats.Conn) {
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1,
	}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)

	go ns.Start()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS server not ready for connections")
	}

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	return ns, nc
}

func TestRegistry_Ingestion(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping integration test: TS_DATABASE_URL not set")
	}

	ns, nc := startTestNatsServer(t)
	defer ns.Shutdown()
	defer nc.Close()

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clear any prior test entries
	_, _ = pool.Exec(ctx, "DELETE FROM agent_registrations")
	_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE name LIKE 'test-%'")
	_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE name LIKE 'test-%'")

	agentRepo := postgres.NewAgentRepository(pool)
	commRepo := postgres.NewCommunityRepository(pool)

	// Create a dummy community
	comm := &model.Community{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        "test-community",
		Description: "For testing",
		Topology:    "hub-spoke",
		Status:      "active",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	err = commRepo.Create(ctx, comm)
	require.NoError(t, err)

	// Create a dummy agent
	agentID := uuid.New()
	ag := &model.Agent{
		ID:          agentID,
		TenantID:    ten.FullName(),
		Name:        "test-agent",
		Description: "For testing",
		Status:      model.AgentStatusAssigned,
		CommunityID: &comm.ID,
		Brain: model.BrainConfig{
			Model:             "gpt-4",
			Temperature:       0.7,
			MaxTokens:         2048,
			Endpoint:          "http://localhost:8080",
			CredentialsSecret: "dummy",
		},
		ShortTermMemory: model.ShortTermMemoryConfig{TTLSeconds: 60},
		LongTermMemory:  model.LongTermMemoryConfig{VectorDimension: 1536},
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	err = agentRepo.Create(ctx, ag)
	require.NoError(t, err)

	logger := zerolog.New(nil)
	redisMock := newMockRedis()
	sharedCache := cache.NewCacheAdapter(redisMock, "keeper")

	// Instantiate the subscriber
	sub := keepernats.NewRegistrySubscriber(nc, agentRepo, sharedCache, logger)
	err = sub.Start(context.Background())
	require.NoError(t, err)
	defer sub.Stop()

	// Compile agent card payload
	card := &agentcard.AgentCard{
		Name:               "test-agent",
		Description:        "For testing",
		URL:                "http://test-agent",
		Version:            "1.0.0",
		Capabilities:       agentcard.AgentCardCapabilities{Streaming: true},
		Authentication:     agentcard.AgentCardAuthentication{Schemes: []string{"Bearer"}},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills:             []agentcard.AgentCardSkill{},
	}

	evt, err := events.NewDomainEvent(
		events.SchemaInfrastructureAgentHeartbeat,
		fmt.Sprintf("agent/%s", ag.ID),
		ten.FullName(),
		card,
	)
	require.NoError(t, err)
	evtBytes, err := json.Marshal(evt)
	require.NoError(t, err)

	msg := nats.NewMsg(fmt.Sprintf("ts.community.%s.agent.%s.heartbeat", comm.ID, ag.ID))
	msg.Data = evtBytes
	msg.Header.Set("X-Tacito-Schema", evt.SchemaRef)
	msg.Header.Set("X-Tacito-Source", evt.Source)
	msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
	msg.Header.Set("X-Tacito-Event-ID", evt.EventID)
	msg.Header.Set("X-Tacito-Occurred", evt.OccurredAt)

	err = nc.PublishMsg(msg)
	require.NoError(t, err)

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// 1. Verify card saved in DB
	fetchedCard, _, err := agentRepo.GetRegistration(ctx, ag.ID, comm.ID)
	require.NoError(t, err)
	assert.Equal(t, card.Name, fetchedCard.Name)
	assert.Equal(t, card.Version, fetchedCard.Version)

	// 2. Verify status updated to running
	agFetched, err := agentRepo.GetByID(ctx, ag.ID)
	require.NoError(t, err)
	assert.Equal(t, model.AgentStatusRunning, agFetched.Status)

	// 3. Verify card cached in Redis
	var cachedCard agentcard.AgentCard
	cacheKey := fmt.Sprintf("communities:%s:agents:%s", comm.ID, ag.ID)
	err = sharedCache.Get(ctx, cacheKey, &cachedCard)
	require.NoError(t, err)
	assert.Equal(t, card.Name, cachedCard.Name)
}
