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

func TestPromptRepository_Lifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ten, _ := tenant.New("test-tenant.com", "")
	ctx := tenant.ContextWithTenant(context.Background(), ten)

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Clean up test records
	_, err = pool.Exec(ctx, "DELETE FROM prompt_collections WHERE name LIKE 'test-%'")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DELETE FROM prompt_templates WHERE name LIKE 'test-%'")
	require.NoError(t, err)

	repo := NewPromptRepository(pool)

	// Create test templates
	pt1 := &model.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-greeting",
		Content:   "Hello {{.Name}}",
		Status:    model.PromptStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	pt2 := &model.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-goodbye",
		Content:   "Goodbye {{.Name}}",
		Status:    model.PromptStatusDraft,
		CreatedAt: time.Now().UTC(),
	}

	t.Run("Create Templates", func(t *testing.T) {
		err := repo.CreateTemplate(ctx, pt1)
		require.NoError(t, err)

		err = repo.CreateTemplate(ctx, pt2)
		require.NoError(t, err)
	})

	t.Run("Get Template By ID", func(t *testing.T) {
		fetched, err := repo.GetTemplateByID(ctx, pt1.ID)
		require.NoError(t, err)
		assert.Equal(t, pt1.ID, fetched.ID)
		assert.Equal(t, pt1.Name, fetched.Name)
		assert.Equal(t, pt1.Content, fetched.Content)
	})

	t.Run("List Templates", func(t *testing.T) {
		list, err := repo.ListTemplates(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, list)

		var names []string
		for _, pt := range list {
			names = append(names, pt.Name)
		}
		assert.Contains(t, names, "test-greeting")
		assert.Contains(t, names, "test-goodbye")
	})

	// Collections test
	collection := &model.PromptCollection{
		ID:          uuid.New(),
		Name:        "test-suite",
		Description: "A suite of prompt templates",
		Templates:   []uuid.UUID{pt1.ID, pt2.ID},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	t.Run("Create Collection", func(t *testing.T) {
		err := repo.CreateCollection(ctx, collection)
		require.NoError(t, err)
	})

	t.Run("Get Collection By ID", func(t *testing.T) {
		fetched, err := repo.GetCollectionByID(ctx, collection.ID)
		require.NoError(t, err)
		assert.Equal(t, collection.ID, fetched.ID)
		assert.Equal(t, collection.Name, fetched.Name)
		assert.ElementsMatch(t, collection.Templates, fetched.Templates)
	})

	t.Run("Update Collection", func(t *testing.T) {
		collection.Description = "Updated test suite description"
		collection.Templates = []uuid.UUID{pt1.ID}
		err := repo.UpdateCollection(ctx, collection)
		require.NoError(t, err)

		fetched, err := repo.GetCollectionByID(ctx, collection.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated test suite description", fetched.Description)
		assert.Equal(t, []uuid.UUID{pt1.ID}, fetched.Templates)
	})

	t.Run("Resolve Collection Prompts (Only Active)", func(t *testing.T) {
		collection.Templates = []uuid.UUID{pt1.ID, pt2.ID}
		err := repo.UpdateCollection(ctx, collection)
		require.NoError(t, err)

		resolved, err := repo.ResolveCollectionPrompts(ctx, collection.ID)
		require.NoError(t, err)
		
		// Should only resolve the active greeting prompt (pt1)
		assert.Len(t, resolved, 1)
		assert.Equal(t, "test-greeting", resolved[0].Name)
		assert.Equal(t, pt1.ID, resolved[0].ID)
	})

	t.Run("Multi-Tenant Isolation", func(t *testing.T) {
		tenA, _ := tenant.New("tenant-a.com", "")
		ctxA := tenant.ContextWithTenant(context.Background(), tenA)

		tenB, _ := tenant.New("tenant-b.com", "")
		ctxB := tenant.ContextWithTenant(context.Background(), tenB)

		// Clean up previous records for these tenants
		_, _ = pool.Exec(ctx, "DELETE FROM prompt_collections WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())
		_, _ = pool.Exec(ctx, "DELETE FROM prompt_templates WHERE tenant_id IN ($1, $2)", tenA.FullName(), tenB.FullName())

		ptA := &model.PromptTemplate{
			ID:        uuid.New(),
			Name:      "test-tenant-scoped",
			Content:   "Content A",
			Status:    model.PromptStatusActive,
			CreatedAt: time.Now().UTC(),
		}

		ptB := &model.PromptTemplate{
			ID:        uuid.New(),
			Name:      "test-tenant-scoped", // same name
			Content:   "Content B",
			Status:    model.PromptStatusActive,
			CreatedAt: time.Now().UTC(),
		}

		// Create under Tenant A
		err := repo.CreateTemplate(ctxA, ptA)
		require.NoError(t, err)

		// Create under Tenant B
		err = repo.CreateTemplate(ctxB, ptB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's template by ID
		_, err = repo.GetTemplateByID(ctxB, ptA.ID)
		assert.Error(t, err)

		// Tenant A should not see Tenant B's template by ID
		_, err = repo.GetTemplateByID(ctxA, ptB.ID)
		assert.Error(t, err)

		// ListTemplates under Tenant A should contain ptA but NOT ptB
		listA, err := repo.ListTemplates(ctxA)
		require.NoError(t, err)
		foundA := false
		foundBInA := false
		for _, pt := range listA {
			if pt.ID == ptA.ID {
				foundA = true
			}
			if pt.ID == ptB.ID {
				foundBInA = true
			}
		}
		assert.True(t, foundA)
		assert.False(t, foundBInA)

		// Collection multi-tenancy
		collA := &model.PromptCollection{
			ID:          uuid.New(),
			Name:        "test-tenant-coll",
			Description: "Tenant A Coll",
			Templates:   []uuid.UUID{ptA.ID},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		collB := &model.PromptCollection{
			ID:          uuid.New(),
			Name:        "test-tenant-coll", // same name
			Description: "Tenant B Coll",
			Templates:   []uuid.UUID{ptB.ID},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		// Create under Tenant A
		err = repo.CreateCollection(ctxA, collA)
		require.NoError(t, err)

		// Create under Tenant B
		err = repo.CreateCollection(ctxB, collB)
		require.NoError(t, err)

		// Tenant B should not see Tenant A's collection by ID
		_, err = repo.GetCollectionByID(ctxB, collA.ID)
		assert.Error(t, err)

		// ListCollections under Tenant A should contain collA but NOT collB
		collsA, err := repo.ListCollections(ctxA)
		require.NoError(t, err)
		foundCollA := false
		foundCollBInA := false
		for _, col := range collsA {
			if col.ID == collA.ID {
				foundCollA = true
			}
			if col.ID == collB.ID {
				foundCollBInA = true
			}
		}
		assert.True(t, foundCollA)
		assert.False(t, foundCollBInA)

		// Clean up
		_ = repo.DeleteCollection(ctxA, collA.ID)
		_ = repo.DeleteCollection(ctxB, collB.ID)
		_ = repo.DeleteTemplate(ctxA, ptA.ID)
		_ = repo.DeleteTemplate(ctxB, ptB.ID)
	})

	t.Run("Delete Collection", func(t *testing.T) {
		err := repo.DeleteCollection(ctx, collection.ID)
		require.NoError(t, err)

		_, err = repo.GetCollectionByID(ctx, collection.ID)
		assert.Error(t, err)
	})

	t.Run("Delete Template", func(t *testing.T) {
		err := repo.DeleteTemplate(ctx, pt1.ID)
		require.NoError(t, err)

		_, err = repo.GetTemplateByID(ctx, pt1.ID)
		assert.Error(t, err)
	})
}
