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

func TestLLMBindingRepository_Lifecycle(t *testing.T) {
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

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants to prevent conflict with parallel tests
		_, _ = pool.Exec(ctx, "DELETE FROM llm_bindings WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		bindingA := &domain.LLMBinding{
			ID:                 uuid.New(),
			Name:               "test-tenant-scoped",
			Description:        "Tenant A GPT-4",
			Provider:           domain.ProviderOpenAI,
			APIBaseURL:         "https://api.openai.com/v1",
			APIKeySecretRef:    "test-secret-key",
			DefaultModel:       "gpt-4",
			Status:             domain.StatusActive,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}

		bindingB := &domain.LLMBinding{
			ID:                 uuid.New(),
			Name:               "test-tenant-scoped", // same name
			Description:        "Tenant B GPT-4",
			Provider:           domain.ProviderOpenAI,
			APIBaseURL:         "https://api.openai.com/v1",
			APIKeySecretRef:    "test-secret-key",
			DefaultModel:       "gpt-4",
			Status:             domain.StatusActive,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.Create(ctxA, bindingA)
		require.NoError(t, err)

		// Create under Tenant B (should succeed because of (tenant_id, name) composite unique constraint!)
		err = repo.Create(ctxB, bindingB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's record by ID
		_, err = repo.GetByID(ctxB, bindingA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's record by ID
		_, err = repo.GetByID(ctxA, bindingB.ID)
		assert.Error(t, err)

		// GetByName under Tenant A should return bindingA
		fetchedA, err := repo.GetByName(ctxA, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, bindingA.ID, fetchedA.ID)

		// GetByName under Tenant B should return bindingB
		fetchedB, err := repo.GetByName(ctxB, "test-tenant-scoped")
		require.NoError(t, err)
		assert.Equal(t, bindingB.ID, fetchedB.ID)

		// List under Tenant A should contain bindingA but NOT bindingB
		listA, err := repo.List(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, b := range listA {
			if b.ID == bindingA.ID {
				foundA = true
			}
			if b.ID == bindingB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Clean up
		_ = repo.Delete(ctxA, bindingA.ID)
		_ = repo.Delete(ctxB, bindingB.ID)
	})

	t.Run("Delete LLM Binding", func(t *testing.T) {
		err := repo.Delete(ctx, binding.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, binding.ID)
		assert.Error(t, err, "should return an error for deleted binding")
	})
}
