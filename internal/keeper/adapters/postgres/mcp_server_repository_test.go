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

func TestMCPServerRepository_Lifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up any test records
	_, err = pool.Exec(ctx, "DELETE FROM mcp_servers WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewMCPServerRepository(pool)

	server := &domain.MCPServer{
		ID:            uuid.New(),
		Name:          "test-sqlite-mcp",
		Description:   "Test SQLite MCP",
		Transport:     domain.TransportStdio,
		Command:       "mcp-sqlite",
		Args:          []string{"--db", "test.db"},
		Env:           map[string]string{"ENV_VAR": "value"},
		Status:        domain.MCPServerStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	t.Run("Create MCP Server", func(t *testing.T) {
		err := repo.Create(ctx, server)
		require.NoError(t, err)
	})

	t.Run("Get MCP Server by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, server.ID)
		require.NoError(t, err)
		assert.Equal(t, server.ID, fetched.ID)
		assert.Equal(t, server.Name, fetched.Name)
		assert.Equal(t, server.Command, fetched.Command)
		assert.Equal(t, server.Args, fetched.Args)
		assert.Equal(t, server.Env, fetched.Env)
	})

	t.Run("Get MCP Server by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, server.Name)
		require.NoError(t, err)
		assert.Equal(t, server.ID, fetched.ID)
	})

	t.Run("List MCP Servers", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, s := range list {
			if s.ID == server.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find the created server in the list")
	})

	t.Run("Update MCP Server", func(t *testing.T) {
		server.Description = "Updated desc"
		server.Args = []string{"--db", "prod.db", "--verbose"}
		server.Env = map[string]string{"ENV_VAR": "newval", "OTHER": "xyz"}
		err := repo.Update(ctx, server)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, server.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated desc", fetched.Description)
		assert.Equal(t, []string{"--db", "prod.db", "--verbose"}, fetched.Args)
		assert.Equal(t, map[string]string{"ENV_VAR": "newval", "OTHER": "xyz"}, fetched.Env)
	})

	t.Run("Delete MCP Server", func(t *testing.T) {
		err := repo.Delete(ctx, server.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, server.ID)
		assert.Error(t, err, "should return an error for deleted server")
	})
}
