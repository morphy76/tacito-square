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
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/agentcard"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryHandler_RequestReply(t *testing.T) {
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

	// Upsert a registration card for the agent
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
	err = agentRepo.UpsertRegistration(ctx, agentID, comm.ID, card)
	require.NoError(t, err)

	logger := zerolog.New(nil)
	redisMock := newMockRedis()
	sharedCache := cache.NewCacheAdapter(redisMock, "keeper")

	// Instantiate the handler
	handler := keepernats.NewRegistryHandler(nc, agentRepo, sharedCache, logger)
	err = handler.Start(context.Background())
	require.NoError(t, err)
	defer handler.Stop()

	// Send NATS request
	subject := fmt.Sprintf("ts.community.%s.registry.request", comm.ID.String())
	reqMsg := &nats.Msg{
		Subject: subject,
		Header:  nats.Header{},
	}
	reqMsg.Header.Set("X-Tacito-Tenant", ten.FullName())

	reply, err := nc.RequestMsg(reqMsg, 2*time.Second)
	require.NoError(t, err)

	var resultCards []*agentcard.AgentCard
	err = json.Unmarshal(reply.Data, &resultCards)
	require.NoError(t, err)

	require.Len(t, resultCards, 1)
	assert.Equal(t, "test-agent", resultCards[0].Name)
	assert.Equal(t, "1.0.0", resultCards[0].Version)
	assert.Equal(t, ten.FullName(), reply.Header.Get("X-Tacito-Tenant"))

	// Verify it was cached in mockRedis under communities:{community_id}:registry
	cacheKey := fmt.Sprintf("ts:keeper:cache:communities:%s:registry", comm.ID.String())
	_, exists := redisMock.store[cacheKey]
	assert.True(t, exists)
}
