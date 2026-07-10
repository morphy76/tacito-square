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

func TestCommunityAssignmentRepository_Lifecycle(t *testing.T) {
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
	_, err = pool.Exec(ctx, "DELETE FROM community_assignments")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM agents WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM communities WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	// Create a test community and agent first (foreign keys)
	commID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO communities (id, tenant_id, name, topology, status, created_at, updated_at) 
		VALUES ($1, $2, 'test-comm-1', 'hub-spoke', 'active', NOW(), NOW())`, commID, ten.FullName())
	require.NoError(t, err)

	agentID1 := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO agents (id, tenant_id, name, brain, short_term_memory, long_term_memory, mcp_clients, status, created_at, updated_at) 
		VALUES ($1, $2, 'test-agent-1', '{"llm_binding_id": "8fa6b8cb-2b8e-4a6c-9a40-d9d1be6596b4"}', '{}', '{}', '[]', 'defined', NOW(), NOW())`, agentID1, ten.FullName())
	require.NoError(t, err)

	agentID2 := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO agents (id, tenant_id, name, brain, short_term_memory, long_term_memory, mcp_clients, status, created_at, updated_at) 
		VALUES ($1, $2, 'test-agent-2', '{"llm_binding_id": "8fa6b8cb-2b8e-4a6c-9a40-d9d1be6596b4"}', '{}', '{}', '[]', 'defined', NOW(), NOW())`, agentID2, ten.FullName())
	require.NoError(t, err)

	repo := NewCommunityAssignmentRepository(pool)

	t.Run("Create assignment successfully", func(t *testing.T) {
		assignment := &model.CommunityAssignment{
			CommunityID: commID,
			AgentID:     agentID1,
			TenantID:    ten.FullName(),
			Role:        model.AgentRoleHub,
			AssignedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, assignment)
		require.NoError(t, err)

		// Verify backward-compatibility trigger synchronized agents.role to "hub"
		var syncedRole string
		err = pool.QueryRow(ctx, "SELECT role FROM agents WHERE id = $1", agentID1).Scan(&syncedRole)
		require.NoError(t, err)
		assert.Equal(t, "hub", syncedRole)
	})

	t.Run("Create duplicate assignment returns unique violation conflict error", func(t *testing.T) {
		assignment := &model.CommunityAssignment{
			CommunityID: commID,
			AgentID:     agentID1,
			TenantID:    ten.FullName(),
			Role:        model.AgentRoleSpoke,
			AssignedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, assignment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already assigned") // or unique constraint message
	})

	t.Run("CountHubs", func(t *testing.T) {
		count, err := repo.CountHubs(ctx, commID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("CountByCommunity", func(t *testing.T) {
		count, err := repo.CountByCommunity(ctx, commID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Create a second spoke assignment", func(t *testing.T) {
		assignment := &model.CommunityAssignment{
			CommunityID: commID,
			AgentID:     agentID2,
			TenantID:    ten.FullName(),
			Role:        model.AgentRoleSpoke,
			AssignedAt:  time.Now().UTC(),
		}

		err := repo.Create(ctx, assignment)
		require.NoError(t, err)

		// Verify backward-compatibility trigger synchronized agents.role to "spoke"
		var syncedRole string
		err = pool.QueryRow(ctx, "SELECT role FROM agents WHERE id = $1", agentID2).Scan(&syncedRole)
		require.NoError(t, err)
		assert.Equal(t, "spoke", syncedRole)
	})

	t.Run("CountByCommunity after second assignment", func(t *testing.T) {
		count, err := repo.CountByCommunity(ctx, commID)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("CountHubs after second assignment", func(t *testing.T) {
		count, err := repo.CountHubs(ctx, commID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("ListByCommunity returns all assignments for community", func(t *testing.T) {
		list, err := repo.ListByCommunity(ctx, commID)
		require.NoError(t, err)
		assert.Len(t, list, 2)

		var hubAssigned, spokeAssigned bool
		for _, a := range list {
			assert.Equal(t, commID, a.CommunityID)
			assert.Equal(t, ten.FullName(), a.TenantID)
			if a.AgentID == agentID1 {
				assert.Equal(t, model.AgentRoleHub, a.Role)
				hubAssigned = true
			} else if a.AgentID == agentID2 {
				assert.Equal(t, model.AgentRoleSpoke, a.Role)
				spokeAssigned = true
			}
		}
		assert.True(t, hubAssigned)
		assert.True(t, spokeAssigned)
	})

	t.Run("Tenant isolation in ListByCommunity", func(t *testing.T) {
		otherTen, _ := tenant.New("other-tenant.com", "")
		otherCtx := tenant.ContextWithTenant(context.Background(), otherTen)

		list, err := repo.ListByCommunity(otherCtx, commID)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("Delete assignment", func(t *testing.T) {
		err := repo.Delete(ctx, commID, agentID1)
		require.NoError(t, err)

		count, err := repo.CountByCommunity(ctx, commID)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Delete non-existent assignment returns not found", func(t *testing.T) {
		err := repo.Delete(ctx, commID, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assignment not found")
	})
}
