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

func TestCommunityRepository_Lifecycle(t *testing.T) {
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
	_, err = pool.Exec(ctx, "DELETE FROM agent_skills")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM agents WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM communities WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewCommunityRepository(pool)

	comm := &model.Community{
		ID:          uuid.New(),
		Name:        "test-community-template",
		Description: "A test community configuration",
		Topology:    model.CommunityTopologyHubSpoke,
		Configuration: map[string]interface{}{
			"max_messages_per_sec": float64(100),
			"enable_monitoring":    true,
		},
		Status:    model.CommunityStatusCreated,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("Create Community", func(t *testing.T) {
		err := repo.Create(ctx, comm)
		require.NoError(t, err)
	})

	t.Run("Get Community by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, comm.ID)
		require.NoError(t, err)
		assert.Equal(t, comm.ID, fetched.ID)
		assert.Equal(t, comm.Name, fetched.Name)
		assert.Equal(t, comm.Topology, fetched.Topology)
		assert.Equal(t, comm.Configuration["max_messages_per_sec"], fetched.Configuration["max_messages_per_sec"])
		assert.Equal(t, comm.Configuration["enable_monitoring"], fetched.Configuration["enable_monitoring"])
		assert.Equal(t, comm.Status, fetched.Status)
	})

	t.Run("Get Community by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, comm.Name)
		require.NoError(t, err)
		assert.Equal(t, comm.ID, fetched.ID)
	})

	t.Run("List Communities", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, c := range list {
			if c.ID == comm.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Update Community", func(t *testing.T) {
		comm.Description = "Updated description"
		comm.Status = model.CommunityStatusActive
		comm.Configuration["max_messages_per_sec"] = float64(200)

		err := repo.Update(ctx, comm)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, comm.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", fetched.Description)
		assert.Equal(t, model.CommunityStatusActive, fetched.Status)
		assert.Equal(t, float64(200), fetched.Configuration["max_messages_per_sec"])
	})

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants
		_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		commA := &model.Community{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped-community",
			Description:   "Tenant A Community",
			Topology:      model.CommunityTopologyHubSpoke,
			Configuration: map[string]interface{}{},
			Status:        model.CommunityStatusCreated,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		commB := &model.Community{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped-community", // same name
			Description:   "Tenant B Community",
			Topology:      model.CommunityTopologyHubSpoke,
			Configuration: map[string]interface{}{},
			Status:        model.CommunityStatusCreated,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, commA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of composite unique constraint)
		err = repo.Create(ctxB, commB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, commA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, commB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return commA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped-community")
		require.NoError(t, err)
		assert.Equal(t, commA.ID, fetchedA.ID)

		// GetByName under Tenant B should return commB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped-community")
		require.NoError(t, err)
		assert.Equal(t, commB.ID, fetchedB.ID)

		// Attempt to create duplicate name under Tenant A (should fail due to unique constraint)
		commADup := &model.Community{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped-community",
			Topology:      model.CommunityTopologyHubSpoke,
			Configuration: map[string]interface{}{},
			Status:        model.CommunityStatusCreated,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err = repo.Create(ctxA, commADup)
		assert.Error(t, err)

		// Cleanup
		_ = repo.Delete(ctxA, commA.ID)
		_ = repo.Delete(ctxB, commB.ID)
	})

	t.Run("Referential Integrity and RESTRICT delete constraint", func(t *testing.T) {
		agentRepo := NewAgentRepository(pool)
		promptRepo := NewPromptRepository(pool)
		skillRepo := NewSkillRepository(pool)

		// Setup prerequisites for agent
		pt := &model.PromptTemplate{
			ID:        uuid.New(),
			Name:      "test-prereq-prompt",
			Content:   "You are an assistant",
			Status:    model.PromptStatusActive,
			CreatedAt: time.Now().UTC(),
		}
		err := promptRepo.CreateTemplate(ctx, pt)
		require.NoError(t, err)

		sk := &model.Skill{
			ID:           uuid.New(),
			Name:         "test-prereq-skill",
			Description:  "Prereq skill",
			Status:       model.SkillStatusActive,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		err = skillRepo.Create(ctx, sk)
		require.NoError(t, err)

		// Create community
		cRestrict := &model.Community{
			ID:            uuid.New(),
			Name:          "test-restrict-community",
			Topology:      model.CommunityTopologyHubSpoke,
			Configuration: map[string]interface{}{},
			Status:        model.CommunityStatusCreated,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err = repo.Create(ctx, cRestrict)
		require.NoError(t, err)

		// Create agent assigned to community
		agent := &model.Agent{
			ID:          uuid.New(),
			Name:        "test-assigned-agent",
			Description: "Assigned agent",
			Brain: model.BrainConfig{
				LLMBindingID: uuid.New(),
				Temperature:  ptrFloat64(0.7),
				MaxTokens:    ptrInt(2048),
			},
			ShortTermMemory: model.ShortTermMemoryConfig{
				KeyNamespace: "test:short",
				TTLSeconds:   3600,
			},
			LongTermMemory: model.LongTermMemoryConfig{
				CollectionName:  "test-long",
				VectorDimension: 1536,
			},
			Skills:         []uuid.UUID{sk.ID},
			PromptTemplate: pt.ID,
			MCPClients:     []model.MCPClientConfig{},
			Status:         model.AgentStatusDefined,
			CommunityID:    &cRestrict.ID,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = agentRepo.Create(ctx, agent)
		require.NoError(t, err)

		// Attempt to delete community (should fail due to ON DELETE RESTRICT foreign key)
		err = repo.Delete(ctx, cRestrict.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "foreign key constraint")

		// Clean up agent first
		err = agentRepo.Delete(ctx, agent.ID)
		require.NoError(t, err)

		// Now deletion should succeed
		err = repo.Delete(ctx, cRestrict.ID)
		require.NoError(t, err)
	})

	t.Run("Topology Mutability Guard", func(t *testing.T) {
		// 1. Create a community
		cMut := &model.Community{
			ID:            uuid.New(),
			TenantID:      ten.FullName(),
			Name:          "test-mut-community",
			Topology:      model.CommunityTopologySingleAgent,
			Status:        model.CommunityStatusCreated,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		err := repo.Create(ctx, cMut)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM communities WHERE id = $1", cMut.ID)
		}()

		// 2. We should be able to update topology to hub-spoke when 0 agents are assigned
		cMut.Topology = model.CommunityTopologyHubSpoke
		err = repo.Update(ctx, cMut)
		assert.NoError(t, err)

		// 3. Create an agent and assign to this community
		agentRepo := NewAgentRepository(pool)
		pt := &model.PromptTemplate{
			ID:        uuid.New(),
			Name:      "test-mut-prompt",
			Content:   "You are an assistant",
			Status:    model.PromptStatusActive,
			CreatedAt: time.Now().UTC(),
		}
		promptRepo := NewPromptRepository(pool)
		err = promptRepo.CreateTemplate(ctx, pt)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM prompt_templates WHERE id = $1", pt.ID)
		}()

		agent := &model.Agent{
			ID:          uuid.New(),
			TenantID:    ten.FullName(),
			Name:        "test-mut-agent",
			Description: "Assigned agent",
			Brain: model.BrainConfig{
				LLMBindingID: uuid.New(),
				Temperature:  ptrFloat64(0.7),
				MaxTokens:    ptrInt(2048),
			},
			ShortTermMemory: model.ShortTermMemoryConfig{
				KeyNamespace: "test:short",
				TTLSeconds:   3600,
			},
			LongTermMemory: model.LongTermMemoryConfig{
				CollectionName:  "test-long",
				VectorDimension: 1536,
			},
			Skills:         []uuid.UUID{},
			PromptTemplate: pt.ID,
			MCPClients:     []model.MCPClientConfig{},
			Status:         model.AgentStatusDefined,
			CommunityID:    nil,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = agentRepo.Create(ctx, agent)
		require.NoError(t, err)
		defer func() {
			_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", agent.ID)
		}()

		err = agentRepo.AssignToCommunity(ctx, agent.ID, cMut.ID)
		require.NoError(t, err)

		// 4. Try to change topology back to single-agent (should fail)
		cMut.Topology = model.CommunityTopologySingleAgent
		err = repo.Update(ctx, cMut)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot change topology of a community with assigned agents")
	})

	t.Run("Delete Community", func(t *testing.T) {
		err := repo.Delete(ctx, comm.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, comm.ID)
		assert.Error(t, err)
	})
}


