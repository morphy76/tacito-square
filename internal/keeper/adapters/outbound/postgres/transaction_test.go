//go:build integration

package postgres

import (
	"context"
	"errors"
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

func TestTransactionRunner_CommitAndRollback(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ten, _ := tenant.New("transaction-test.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up previous runs
	_, err = pool.Exec(ctx, "DELETE FROM agent_skills")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM agents WHERE name LIKE 'tx-%'")
	require.NoError(t, err)

	txRunner := NewTransactionRunner(pool)
	repo := NewAgentRepository(pool)

	agent := &model.Agent{
		ID:          uuid.New(),
		Name:        "tx-agent-success",
		Description: "A transaction test agent",
		Brain: model.BrainConfig{
			Model:             "gpt-4o",
			Temperature:       0.7,
			MaxTokens:         2048,
			Endpoint:          "https://api.openai.com/v1",
			CredentialsSecret: "dummy-secret",
		},
		ShortTermMemory: model.ShortTermMemoryConfig{
			KeyNamespace: "tx:short",
			TTLSeconds:   3600,
		},
		LongTermMemory: model.LongTermMemoryConfig{
			CollectionName:  "tx-long",
			VectorDimension: 1536,
		},
		Status:    model.AgentStatusDefined,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("Successful Transaction Commits", func(t *testing.T) {
		err := txRunner.RunInTransaction(ctx, func(txCtx context.Context) error {
			// Inside the transactional context txCtx
			createErr := repo.Create(txCtx, agent)
			if createErr != nil {
				return createErr
			}
			return nil
		})
		require.NoError(t, err)

		// Verify it was committed and can be fetched from standard pool
		fetched, getErr := repo.GetByID(ctx, agent.ID)
		require.NoError(t, getErr)
		assert.Equal(t, agent.ID, fetched.ID)
		assert.Equal(t, agent.Name, fetched.Name)
	})

	t.Run("Failed Transaction Rolls Back", func(t *testing.T) {
		agentFailed := &model.Agent{
			ID:          uuid.New(),
			Name:        "tx-agent-rollback",
			Description: "This agent should be rolled back",
			Brain:       agent.Brain,
			ShortTermMemory: agent.ShortTermMemory,
			LongTermMemory:  agent.LongTermMemory,
			Status:      model.AgentStatusDefined,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		err := txRunner.RunInTransaction(ctx, func(txCtx context.Context) error {
			createErr := repo.Create(txCtx, agentFailed)
			if createErr != nil {
				return createErr
			}
			// Trigger a forced failure inside the transaction callback
			return errors.New("forced transaction rollback failure")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forced transaction rollback failure")

		// Verify the record was not inserted and returns a not found error
		_, getErr := repo.GetByID(ctx, agentFailed.ID)
		assert.Error(t, getErr)
	})
}
