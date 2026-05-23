//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillRepository_Lifecycle(t *testing.T) {
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
	_, err = pool.Exec(ctx, "DELETE FROM skills WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM mcp_servers WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewSkillRepository(pool)
	mcpRepo := NewMCPServerRepository(pool)

	// Create test MCP servers first
	mcp1 := &domain.MCPServer{
		ID:        uuid.New(),
		Name:      "test-mcp-1",
		Transport: domain.TransportStdio,
		Command:   "node",
		Status:    domain.MCPServerStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = mcpRepo.Create(ctx, mcp1)
	require.NoError(t, err)

	mcp2 := &domain.MCPServer{
		ID:        uuid.New(),
		Name:      "test-mcp-2",
		Transport: domain.TransportSSE,
		URL:       "http://localhost/events",
		Status:    domain.MCPServerStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = mcpRepo.Create(ctx, mcp2)
	require.NoError(t, err)

	skill := &domain.Skill{
		ID:           uuid.New(),
		Name:         "test-web-search",
		Description:  "Test skill",
		MCPServers:   []uuid.UUID{mcp1.ID, mcp2.ID},
		AllowedTools: []string{"search_google", "fetch_url"},
		DeniedTools:  []string{"format_disk"},
		Status:       domain.SkillStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	t.Run("Create Skill", func(t *testing.T) {
		err := repo.Create(ctx, skill)
		require.NoError(t, err)
	})

	t.Run("Get Skill by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, skill.ID)
		require.NoError(t, err)
		assert.Equal(t, skill.ID, fetched.ID)
		assert.Equal(t, skill.Name, fetched.Name)
		assert.ElementsMatch(t, skill.MCPServers, fetched.MCPServers)
		assert.Equal(t, skill.AllowedTools, fetched.AllowedTools)
		assert.Equal(t, skill.DeniedTools, fetched.DeniedTools)
	})

	t.Run("Get Skill by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, skill.Name)
		require.NoError(t, err)
		assert.Equal(t, skill.ID, fetched.ID)
	})

	t.Run("List Skills", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, s := range list {
			if s.ID == skill.ID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("Update Skill", func(t *testing.T) {
		skill.Description = "Updated skill desc"
		skill.MCPServers = []uuid.UUID{mcp1.ID} // remove mcp2
		err := repo.Update(ctx, skill)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, skill.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated skill desc", fetched.Description)
		assert.Equal(t, []uuid.UUID{mcp1.ID}, fetched.MCPServers)
	})

	t.Run("Agent-Skill Association", func(t *testing.T) {
		agentRepo := NewAgentRepository(pool)
		agentID := uuid.New()
		agent := &domain.Agent{
			ID:          agentID,
			Name:        "test-agent-skill-assoc",
			Description: "Test agent for skill association",
			Brain: domain.BrainConfig{
				Model:             "gpt-4o",
				Temperature:       0.7,
				MaxTokens:         2048,
				Endpoint:          "https://api.openai.com/v1",
				CredentialsSecret: "my-secret-key",
			},
			ShortTermMemory: domain.ShortTermMemoryConfig{
				KeyNamespace: "test:short",
				TTLSeconds:   3600,
			},
			LongTermMemory: domain.LongTermMemoryConfig{
				CollectionName:  "test-long",
				VectorDimension: 1536,
			},
			Skills:         []uuid.UUID{},
			MCPClients:     []domain.MCPClientConfig{},
			Status:         domain.AgentStatusDefined,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err := agentRepo.Create(ctx, agent)
		require.NoError(t, err)
		defer func() {
			_ = agentRepo.Delete(ctx, agentID)
		}()

		// Attach
		err = repo.AttachSkillToAgent(ctx, agentID, skill.ID)
		require.NoError(t, err)

		// List by agent
		skills, err := repo.ListSkillsByAgent(ctx, agentID)
		require.NoError(t, err)
		assert.Len(t, skills, 1)
		assert.Equal(t, skill.ID, skills[0].ID)

		// Detach
		err = repo.DetachSkillFromAgent(ctx, agentID, skill.ID)
		require.NoError(t, err)

		// List again
		skills, err = repo.ListSkillsByAgent(ctx, agentID)
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants to prevent conflict with parallel tests
		_, _ = pool.Exec(ctx, "DELETE FROM skills WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		skillA := &domain.Skill{
			ID:           uuid.New(),
			Name:         "test-tenant-scoped",
			Description:  "Tenant A Skill",
			MCPServers:   []uuid.UUID{},
			AllowedTools: []string{"search_google"},
			DeniedTools:  []string{"format_disk"},
			Status:       domain.SkillStatusActive,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		skillB := &domain.Skill{
			ID:           uuid.New(),
			Name:         "test-tenant-scoped", // same name
			Description:  "Tenant B Skill",
			MCPServers:   []uuid.UUID{},
			AllowedTools: []string{"search_google"},
			DeniedTools:  []string{"format_disk"},
			Status:       domain.SkillStatusActive,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, skillA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of (tenant_id, name) composite unique constraint!)
		err = repo.Create(ctxB, skillB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, skillA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, skillB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return skillA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, skillA.ID, fetchedA.ID)

		// GetByName under Tenant B should return skillB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, skillB.ID, fetchedB.ID)

		// List under Tenant A should contain skillA but NOT skillB
		listA, err := repo.List(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, s := range listA {
			if s.ID == skillA.ID {
				foundA = true
			}
			if s.ID == skillB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Agent-Skill association isolation
		agentRepo := NewAgentRepository(pool)
		agentID := uuid.New()
		agentB := &domain.Agent{
			ID:          agentID,
			Name:        "test-tenant-b-agent",
			Description: "Tenant B Agent",
			Brain: domain.BrainConfig{
				Model:             "gpt-4o",
				Temperature:       0.7,
				MaxTokens:         2048,
				Endpoint:          "https://api.openai.com/v1",
				CredentialsSecret: "my-secret-key",
			},
			ShortTermMemory: domain.ShortTermMemoryConfig{
				KeyNamespace: "test:short",
				TTLSeconds:   3600,
			},
			LongTermMemory: domain.LongTermMemoryConfig{
				CollectionName:  "test-long",
				VectorDimension: 1536,
			},
			Skills:         []uuid.UUID{},
			MCPClients:     []domain.MCPClientConfig{},
			Status:         domain.AgentStatusDefined,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		err = agentRepo.Create(ctxB, agentB)
		require.NoError(t, err)
		defer func() {
			_ = agentRepo.Delete(ctxB, agentID)
		}()

		// Detach under Tenant B for Tenant A's skill should have no effect
		err = repo.AttachSkillToAgent(ctxB, agentID, skillA.ID) // should fail/do nothing because skillA is not in Tenant B
		require.NoError(t, err)

		// List by agent under Tenant B should be empty
		skillsB, err := repo.ListSkillsByAgent(ctxB, agentID)
		require.NoError(t, err)
		assert.Empty(t, skillsB)

		// Clean up
		_ = repo.Delete(ctxA, skillA.ID)
		_ = repo.Delete(ctxB, skillB.ID)
	})

	t.Run("Delete Skill", func(t *testing.T) {
		err := repo.Delete(ctx, skill.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, skill.ID)
		assert.Error(t, err)
	})
}
