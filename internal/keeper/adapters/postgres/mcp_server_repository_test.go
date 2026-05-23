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

func TestMCPServerRepository_Lifecycle(t *testing.T) {
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

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants to prevent conflict with parallel tests
		_, _ = pool.Exec(ctx, "DELETE FROM mcp_servers WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		serverA := &domain.MCPServer{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped",
			Description:   "Tenant A MCP",
			Transport:     domain.TransportStdio,
			Command:       "mcp-a",
			Status:        domain.MCPServerStatusActive,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		serverB := &domain.MCPServer{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped", // same name
			Description:   "Tenant B MCP",
			Transport:     domain.TransportStdio,
			Command:       "mcp-b",
			Status:        domain.MCPServerStatusActive,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, serverA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of (tenant_id, name) composite unique constraint!)
		err = repo.Create(ctxB, serverB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, serverA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, serverB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return serverA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, serverA.ID, fetchedA.ID)

		// GetByName under Tenant B should return serverB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, serverB.ID, fetchedB.ID)

		// List under Tenant A should contain serverA but NOT serverB
		listA, err := repo.List(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, s := range listA {
			if s.ID == serverA.ID {
				foundA = true
			}
			if s.ID == serverB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Clean up
		_ = repo.Delete(ctxA, serverA.ID)
		_ = repo.Delete(ctxB, serverB.ID)
	})

	t.Run("Delete MCP Server", func(t *testing.T) {
		err := repo.Delete(ctx, server.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, server.ID)
		assert.Error(t, err, "should return an error for deleted server")
	})
}
