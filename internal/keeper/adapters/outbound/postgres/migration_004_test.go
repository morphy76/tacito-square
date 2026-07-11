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

// TestMigration_PromptAssociationSchema verifies the schema introduced by
// migration 00004_prompt_collection_and_association.sql.
func TestMigration_PromptAssociationSchema(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ten, _ := tenant.New("migration-test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	t.Run("prompt_versions table exists", func(t *testing.T) {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'prompt_versions'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "expected table prompt_versions to exist")
	})

	t.Run("agent_prompts table exists", func(t *testing.T) {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'agent_prompts'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "expected table agent_prompts to exist")
	})

	t.Run("agent_prompt_collections table exists", func(t *testing.T) {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = 'agent_prompt_collections'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "expected table agent_prompt_collections to exist")
	})

	t.Run("agents table no longer has prompt_template column", func(t *testing.T) {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'agents' AND column_name = 'prompt_template'
			)
		`).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "expected column prompt_template to be removed from agents table")
	})

	t.Run("system prompt template has a seeded version 1 in prompt_versions", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM prompt_versions
			WHERE prompt_id = 'ffffffff-0000-0000-0000-000000000001'
			AND version_number = 1
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "expected system hub prompt template to have version 1 seeded")
	})

	t.Run("agent_prompts backfill for pre-existing agents with prompt_template", func(t *testing.T) {
		// Create a prompt template
		promptID := uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO prompt_templates (id, tenant_id, name, content, status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, promptID, ten.FullName(), "mig-test-prompt-"+promptID.String(), "system content", "active", time.Now().UTC())
		require.NoError(t, err)

		// Create a version 1 record as migration would have done
		_, err = pool.Exec(ctx, `
			INSERT INTO prompt_versions (id, prompt_id, version_number, content_snapshot, created_at)
			VALUES ($1, $2, 1, $3, $4)
		`, uuid.New(), promptID, "system content", time.Now().UTC())
		require.NoError(t, err)

		// Create an agent and manually link to agent_prompts (since prompt_template column is gone)
		agentID := uuid.New()
		_, err = pool.Exec(ctx, `
			INSERT INTO agents (id, tenant_id, name, brain, short_term_memory, long_term_memory, mcp_clients, status, tier, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			agentID, ten.FullName(), "mig-test-agent-"+agentID.String(),
			`{"llm_binding_id": "00000000-0000-0000-0000-000000000001"}`,
			`{"key_namespace":"test","ttl_seconds":3600}`,
			`{"collection_name":"","vector_dimension":0}`,
			`[]`,
			model.AgentStatusDefined, "", time.Now().UTC(), time.Now().UTC(),
		)
		require.NoError(t, err)

		// Link agent to prompt via agent_prompts
		_, err = pool.Exec(ctx, `
			INSERT INTO agent_prompts (agent_id, prompt_template_id, position)
			VALUES ($1, $2, 0)
		`, agentID, promptID)
		require.NoError(t, err)

		// Verify the link is readable
		var linkedPromptID uuid.UUID
		err = pool.QueryRow(ctx, `
			SELECT prompt_template_id FROM agent_prompts
			WHERE agent_id = $1 AND position = 0
		`, agentID).Scan(&linkedPromptID)
		require.NoError(t, err)
		assert.Equal(t, promptID, linkedPromptID)

		// Cleanup
		_, _ = pool.Exec(ctx, "DELETE FROM agent_prompts WHERE agent_id = $1", agentID)
		_, _ = pool.Exec(ctx, "DELETE FROM agents WHERE id = $1", agentID)
		_, _ = pool.Exec(ctx, "DELETE FROM prompt_versions WHERE prompt_id = $1", promptID)
		_, _ = pool.Exec(ctx, "DELETE FROM prompt_templates WHERE id = $1", promptID)
	})

	t.Run("unique constraint on agent_prompts primary key", func(t *testing.T) {
		var constraintExists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = 'public'
				AND table_name = 'agent_prompts'
				AND constraint_type = 'PRIMARY KEY'
			)
		`).Scan(&constraintExists)
		require.NoError(t, err)
		assert.True(t, constraintExists)
	})

	t.Run("unique constraint on agent_prompt_collections primary key", func(t *testing.T) {
		var constraintExists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = 'public'
				AND table_name = 'agent_prompt_collections'
				AND constraint_type = 'PRIMARY KEY'
			)
		`).Scan(&constraintExists)
		require.NoError(t, err)
		assert.True(t, constraintExists)
	})

	t.Run("unique constraint on prompt_versions prompt_id+version_number", func(t *testing.T) {
		var constraintExists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = 'public'
				AND table_name = 'prompt_versions'
				AND constraint_name = 'unique_prompt_id_version'
			)
		`).Scan(&constraintExists)
		require.NoError(t, err)
		assert.True(t, constraintExists)
	})
}
