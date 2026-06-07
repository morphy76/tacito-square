//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentRepository_AssignmentLifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up any test records
	_, err = pool.Exec(ctx, "DELETE FROM agents WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM communities WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewAgentRepository(pool)
	commRepo := NewCommunityRepository(pool)

	// Prerequisite: Create Community
	comm := &model.Community{
		ID:            uuid.New(),
		TenantID:      ten.FullName(),
		Name:          "test-community",
		Description:   "Test community for assignment",
		Topology:      model.CommunityTopologyHubSpoke,
		Status:        "active",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	err = commRepo.Create(ctx, comm)
	require.NoError(t, err)

	// Prerequisite: Create Agent
	agent := &model.Agent{
		ID:            uuid.New(),
		TenantID:      ten.FullName(),
		Name:          "test-assign-agent",
		Description:   "Test agent",
		Brain: model.BrainConfig{
			Model:             "gpt-4o",
			Temperature:       0.7,
			MaxTokens:         2048,
			Endpoint:          "https://api.openai.com/v1",
			CredentialsSecret: "my-secret-key",
		},
		ShortTermMemory: model.ShortTermMemoryConfig{
			KeyNamespace: "test:short",
			TTLSeconds:   3600,
		},
		LongTermMemory: model.LongTermMemoryConfig{
			CollectionName:  "test-long",
			VectorDimension: 1536,
		},
		MCPClients: []model.MCPClientConfig{},
		Status:     model.AgentStatusDefined,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	err = repo.Create(ctx, agent)
	require.NoError(t, err)

	t.Run("Assign Agent to Community Successfully", func(t *testing.T) {
		err := repo.AssignToCommunity(ctx, agent.ID, comm.ID)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, model.AgentStatusPending, fetched.Status)
		require.NotNil(t, fetched.CommunityID)
		assert.Equal(t, comm.ID, *fetched.CommunityID)
	})

	t.Run("Double Assignment Guard (409 logic check)", func(t *testing.T) {
		// Try to assign again to the same community — should fail
		err := repo.AssignToCommunity(ctx, agent.ID, comm.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned to community")

		// Create another community and try to assign
		comm2 := &model.Community{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-community-2",
			Description:   "Test community 2",
			Topology:      model.CommunityTopologyHubSpoke,
			Status:        "active",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err = commRepo.Create(ctx, comm2)
		require.NoError(t, err)

		err = repo.AssignToCommunity(ctx, agent.ID, comm2.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned to community")

		// Cleanup comm2
		_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE id = $1", comm2.ID)
	})

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Create community and agent under Tenant B
		commB := &model.Community{
			ID:            uuid.New(),
			TenantID:      tenB.FullName(),
			Name:          "test-community-b",
			Description:   "Tenant B Community",
			Topology:      model.CommunityTopologyHubSpoke,
			Status:        "active",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err = commRepo.Create(ctxB, commB)
		require.NoError(t, err)

		agentB := &model.Agent{
			ID:            uuid.New(),
			TenantID:      tenB.FullName(),
			Name:          "test-agent-b",
			Description:   "Tenant B Agent",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctxB, agentB)
		require.NoError(t, err)

		// Attempting to assign Tenant B's agent to Tenant A's community from Tenant B's context should fail (comm not found)
		err = repo.AssignToCommunity(ctxB, agentB.ID, comm.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "community not found")

		// Attempting to assign Tenant A's agent to Tenant B's community from Tenant B's context should fail (agent not found)
		err = repo.AssignToCommunity(ctxB, agent.ID, commB.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")

		// Cleanup Tenant B records
		_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE tenant_id = $1", tenB.FullName())
		_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE tenant_id = $1", tenB.FullName())
	})

	t.Run("Unassign Agent from Community Successfully", func(t *testing.T) {
		err := repo.UnassignFromCommunity(ctx, agent.ID, comm.ID)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, model.AgentStatusDefined, fetched.Status)
		assert.Nil(t, fetched.CommunityID)
	})

	t.Run("Unassign Non-Assigned Agent Guard", func(t *testing.T) {
		err := repo.UnassignFromCommunity(ctx, agent.ID, comm.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not assigned to community")
	})

	t.Run("Single-Agent Community Assignment Limit", func(t *testing.T) {
		// 1. Create a single-agent community
		singleComm := &model.Community{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-single-agent-comm",
			Topology:      model.CommunityTopologySingleAgent,
			Status:        "active",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err := commRepo.Create(ctx, singleComm)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE id = $1", singleComm.ID)
		}()

		// 2. Create agent 1 and assign
		a1 := &model.Agent{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-a1",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, a1)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", a1.ID)
		}()

		err = repo.AssignToCommunity(ctx, a1.ID, singleComm.ID)
		require.NoError(t, err)

		// 3. Create agent 2 and attempt to assign (should fail)
		a2 := &model.Agent{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-a2",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, a2)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", a2.ID)
		}()

		err = repo.AssignToCommunity(ctx, a2.ID, singleComm.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "single-agent topology cannot have more than one agent assigned")
	})

	t.Run("Hub-Spoke Community Hub Uniqueness Limit", func(t *testing.T) {
		// 1. Create a hub-spoke community
		hubSpokeComm := &model.Community{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-hubspoke-comm",
			Topology:      model.CommunityTopologyHubSpoke,
			Status:        "active",
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err := commRepo.Create(ctx, hubSpokeComm)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE id = $1", hubSpokeComm.ID)
		}()

		// 2. Create hub agent 1 and assign
		h1 := &model.Agent{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-h1",
			Role:          "hub",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, h1)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", h1.ID)
		}()

		err = repo.AssignToCommunity(ctx, h1.ID, hubSpokeComm.ID)
		require.NoError(t, err)

		// 3. Create hub agent 2 and attempt to assign (should fail)
		h2 := &model.Agent{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-h2",
			Role:          "hub",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, h2)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", h2.ID)
		}()

		err = repo.AssignToCommunity(ctx, h2.ID, hubSpokeComm.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hub-spoke topology cannot have more than one hub agent assigned")

		// 4. Create spoke agent and assign (should succeed)
		spoke1 := &model.Agent{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-spoke1",
			Role:          "spoke",
			Brain:         agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}
		err = repo.Create(ctx, spoke1)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", spoke1.ID)
		}()

		err = repo.AssignToCommunity(ctx, spoke1.ID, hubSpokeComm.ID)
		assert.NoError(t, err)
	})

	// Cleanup all records
	_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", agent.ID)
	_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE id = $1", comm.ID)
}
