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

func TestMCPClientRepository_Lifecycle(t *testing.T) {
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
	_, err = pool.Exec(ctx, "DELETE FROM mcp_clients WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewMCPClientRepository(pool)

	clientObj := &model.MCPClient{
		ID:            uuid.New(),
		Name:          "test-sqlite-mcp",
		Description:   "Test SQLite MCP",
		Transport:     model.TransportStdio,
		Command:       "mcp-sqlite",
		Args:          []string{"--db", "test.db"},
		Env:           map[string]string{"ENV_VAR": "value"},
		Status:        model.MCPClientStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	t.Run("Create MCP Client", func(t *testing.T) {
		err := repo.Create(ctx, clientObj)
		require.NoError(t, err)
	})

	t.Run("Get MCP Client by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, clientObj.ID)
		require.NoError(t, err)
		assert.Equal(t, clientObj.ID, fetched.ID)
		assert.Equal(t, clientObj.Name, fetched.Name)
		assert.Equal(t, clientObj.Command, fetched.Command)
		assert.Equal(t, clientObj.Args, fetched.Args)
		assert.Equal(t, clientObj.Env, fetched.Env)
	})

	t.Run("Get MCP Client by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, clientObj.Name)
		require.NoError(t, err)
		assert.Equal(t, clientObj.ID, fetched.ID)
	})

	t.Run("List MCP Clients", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, s := range list {
			if s.ID == clientObj.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find the created client in the list")
	})

	t.Run("Update MCP Client", func(t *testing.T) {
		clientObj.Description = "Updated desc"
		clientObj.Args = []string{"--db", "prod.db", "--verbose"}
		clientObj.Env = map[string]string{"ENV_VAR": "newval", "OTHER": "xyz"}
		err := repo.Update(ctx, clientObj)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, clientObj.ID)
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
		_, _ = pool.Exec(ctx, "DELETE FROM mcp_clients WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		clientA := &model.MCPClient{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped",
			Description:   "Tenant A MCP",
			Transport:     model.TransportStdio,
			Command:       "mcp-a",
			Status:        model.MCPClientStatusActive,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		clientB := &model.MCPClient{
			ID:            uuid.New(),
			Name:          "test-tenant-scoped", // same name
			Description:   "Tenant B MCP",
			Transport:     model.TransportStdio,
			Command:       "mcp-b",
			Status:        model.MCPClientStatusActive,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, clientA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of (tenant_id, name) composite unique constraint!)
		err = repo.Create(ctxB, clientB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, clientA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, clientB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return clientA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, clientA.ID, fetchedA.ID)

		// GetByName under Tenant B should return clientB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, clientB.ID, fetchedB.ID)

		// List under Tenant A should contain clientA but NOT clientB
		listA, err := repo.List(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, s := range listA {
			if s.ID == clientA.ID {
				foundA = true
			}
			if s.ID == clientB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Clean up
		_ = repo.Delete(ctxA, clientA.ID)
		_ = repo.Delete(ctxB, clientB.ID)
	})

	t.Run("Delete MCP Client", func(t *testing.T) {
		err := repo.Delete(ctx, clientObj.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, clientObj.ID)
		assert.Error(t, err, "should return an error for deleted client")
	})
}
