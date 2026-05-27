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

func TestAgentRepository_Lifecycle(t *testing.T) {
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
	_, err = pool.Exec(ctx, "DELETE FROM skills WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM prompt_templates WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewAgentRepository(pool)
	skillRepo := NewSkillRepository(pool)
	promptRepo := NewPromptRepository(pool)

	// Create a prerequisite prompt template
	pt := &model.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-system-prompt",
		Content:   "You are an assistant",
		Status:    model.PromptStatusActive,
		CreatedAt: time.Now().UTC(),
	}
	err = promptRepo.CreateTemplate(ctx, pt)
	require.NoError(t, err)

	// Create a prerequisite skill
	sk := &model.Skill{
		ID:           uuid.New(),
		Name:         "test-basic-skill",
		Description:  "Prereq skill",
		Status:       model.SkillStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	err = skillRepo.Create(ctx, sk)
	require.NoError(t, err)

	agent := &model.Agent{
		ID:          uuid.New(),
		Name:        "test-agent-template",
		Description: "A test agent template configuration",
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
		Skills:         []uuid.UUID{sk.ID},
		PromptTemplate: pt.ID,
		MCPClients: []model.MCPClientConfig{
			{
				ServerID: uuid.New(),
			},
		},
		Status:    model.AgentStatusDefined,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("Create Agent", func(t *testing.T) {
		err := repo.Create(ctx, agent)
		require.NoError(t, err)
	})

	t.Run("Get Agent by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, fetched.ID)
		assert.Equal(t, agent.Name, fetched.Name)
		assert.Equal(t, agent.Brain.Model, fetched.Brain.Model)
		assert.Equal(t, agent.ShortTermMemory.KeyNamespace, fetched.ShortTermMemory.KeyNamespace)
		assert.Equal(t, agent.LongTermMemory.CollectionName, fetched.LongTermMemory.CollectionName)
		assert.Equal(t, agent.PromptTemplate, fetched.PromptTemplate)
		assert.Equal(t, agent.Status, fetched.Status)
		assert.Equal(t, len(agent.Skills), len(fetched.Skills))
		if len(fetched.Skills) > 0 {
			assert.Equal(t, agent.Skills[0], fetched.Skills[0])
		}
		assert.Equal(t, len(agent.MCPClients), len(fetched.MCPClients))
		if len(fetched.MCPClients) > 0 {
			assert.Equal(t, agent.MCPClients[0].ServerID, fetched.MCPClients[0].ServerID)
		}
	})

	t.Run("Get Agent by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, agent.Name)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, fetched.ID)
	})

	t.Run("List Agents", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, a := range list {
			if a.ID == agent.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Update Agent", func(t *testing.T) {
		agent.Description = "Updated description"
		agent.Brain.Temperature = 0.9
		agent.Status = model.AgentStatusActive
		agent.Skills = []uuid.UUID{} // Clear skills
		agent.MCPClients = append(agent.MCPClients, model.MCPClientConfig{ServerID: uuid.New()})

		err := repo.Update(ctx, agent)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", fetched.Description)
		assert.Equal(t, 0.9, fetched.Brain.Temperature)
		assert.Equal(t, model.AgentStatusActive, fetched.Status)
		assert.Empty(t, fetched.Skills)
		assert.Len(t, fetched.MCPClients, 2)
	})

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants to prevent conflict with parallel tests
		_, _ = pool.Exec(ctx, "DELETE FROM agent_skills")
		_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		agentA := &model.Agent{
			ID:          uuid.New(),
			Name:        "test-tenant-scoped-agent",
			Description: "Tenant A Agent",
			Brain:       agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}

		agentB := &model.Agent{
			ID:          uuid.New(),
			Name:        "test-tenant-scoped-agent", // same name
			Description: "Tenant B Agent",
			Brain:       agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			MCPClients:      []model.MCPClientConfig{},
			Status:          model.AgentStatusDefined,
			CreatedAt:       time.Now().UTC(),
			UpdatedAt:       time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, agentA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of composite unique constraint)
		err = repo.Create(ctxB, agentB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, agentA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, agentB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return agentA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped-agent")
		require.NoError(t, err)
		assert.Equal(t, agentA.ID, fetchedA.ID)

		// GetByName under Tenant B should return agentB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped-agent")
		require.NoError(t, err)
		assert.Equal(t, agentB.ID, fetchedB.ID)

		// List under Tenant A should contain agentA but NOT agentB
		listA, err := repo.List(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, a := range listA {
			if a.ID == agentA.ID {
				foundA = true
			}
			if a.ID == agentB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Clean up
		_ = repo.Delete(ctxA, agentA.ID)
		_ = repo.Delete(ctxB, agentB.ID)
	})

	t.Run("Delete Agent", func(t *testing.T) {
		err := repo.Delete(ctx, agent.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, agent.ID)
		assert.Error(t, err)
	})
}
