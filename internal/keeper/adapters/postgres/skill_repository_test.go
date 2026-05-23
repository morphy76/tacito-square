package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillRepository_Lifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ctx := context.Background()
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
		agentID := uuid.New()

		// Attach
		err := repo.AttachSkillToAgent(ctx, agentID, skill.ID)
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

	t.Run("Delete Skill", func(t *testing.T) {
		err := repo.Delete(ctx, skill.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, skill.ID)
		assert.Error(t, err)
	})
}
