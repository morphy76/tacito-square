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

func TestLLMBindingRepository_Lifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up any test records
	_, err = pool.Exec(ctx, "DELETE FROM llm_bindings WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewLLMBindingRepository(pool)

	binding := &domain.LLMBinding{
		ID:                 uuid.New(),
		Name:               "test-openai-gpt4o",
		Description:        "Test GPT-4o Binding",
		Provider:           domain.ProviderOpenAI,
		APIBaseURL:         "https://api.openai.com/v1",
		APIKeySecretRef:    "test-secret-key",
		DefaultModel:       "gpt-4o",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   2048,
		TimeoutSeconds:     30,
		Status:             domain.StatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	t.Run("Create LLM Binding", func(t *testing.T) {
		err := repo.Create(ctx, binding)
		require.NoError(t, err)
	})

	t.Run("Get LLM Binding by ID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, binding.ID)
		require.NoError(t, err)
		assert.Equal(t, binding.ID, fetched.ID)
		assert.Equal(t, binding.Name, fetched.Name)
		assert.Equal(t, binding.Provider, fetched.Provider)
		assert.Equal(t, binding.APIBaseURL, fetched.APIBaseURL)
	})

	t.Run("Get LLM Binding by Name", func(t *testing.T) {
		fetched, err := repo.GetByName(ctx, binding.Name)
		require.NoError(t, err)
		assert.Equal(t, binding.ID, fetched.ID)
	})

	t.Run("List LLM Bindings", func(t *testing.T) {
		list, err := repo.List(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		found := false
		for _, b := range list {
			if b.ID == binding.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "should find the created binding in the list")
	})

	t.Run("Update LLM Binding", func(t *testing.T) {
		binding.Description = "Updated description"
		binding.DefaultTemperature = 0.5
		err := repo.Update(ctx, binding)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, binding.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated description", fetched.Description)
		assert.Equal(t, 0.5, fetched.DefaultTemperature)
	})

	t.Run("Delete LLM Binding", func(t *testing.T) {
		err := repo.Delete(ctx, binding.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, binding.ID)
		assert.Error(t, err, "should return an error for deleted binding")
	})
}
