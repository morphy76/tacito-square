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

func TestPromptRepository_Lifecycle(t *testing.T) {
	dbURL := os.Getenv("TS_DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping PostgreSQL integration test: TS_DATABASE_URL not set")
	}

	ctx := context.Background()
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
	pt1_v1 := &domain.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-greeting",
		Content:   "Hello {{.Name}}",
		Role:      domain.PromptRoleSystem,
		Version:   1,
		Status:    domain.PromptStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	pt1_v2 := &domain.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-greeting",
		Content:   "Hello and welcome {{.Name}}",
		Role:      domain.PromptRoleSystem,
		Version:   2,
		Status:    domain.PromptStatusActive,
		CreatedAt: time.Now().UTC(),
	}

	pt2_v1 := &domain.PromptTemplate{
		ID:        uuid.New(),
		Name:      "test-goodbye",
		Content:   "Goodbye {{.Name}}",
		Role:      domain.PromptRoleSystem,
		Version:   1,
		Status:    domain.PromptStatusDraft,
		CreatedAt: time.Now().UTC(),
	}

	t.Run("Create Templates", func(t *testing.T) {
		err := repo.CreateTemplate(ctx, pt1_v1)
		require.NoError(t, err)

		err = repo.CreateTemplate(ctx, pt1_v2)
		require.NoError(t, err)

		err = repo.CreateTemplate(ctx, pt2_v1)
		require.NoError(t, err)
	})

	t.Run("Get Template By ID", func(t *testing.T) {
		fetched, err := repo.GetTemplateByID(ctx, pt1_v1.ID)
		require.NoError(t, err)
		assert.Equal(t, pt1_v1.ID, fetched.ID)
		assert.Equal(t, pt1_v1.Name, fetched.Name)
		assert.Equal(t, pt1_v1.Content, fetched.Content)
		assert.Equal(t, pt1_v1.Version, fetched.Version)
	})

	t.Run("Get Latest Template By Name", func(t *testing.T) {
		fetched, err := repo.GetLatestTemplateByName(ctx, "test-greeting")
		require.NoError(t, err)
		assert.Equal(t, pt1_v2.ID, fetched.ID)
		assert.Equal(t, 2, fetched.Version)
	})

	t.Run("List Templates (Latest)", func(t *testing.T) {
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

	t.Run("List Template Versions", func(t *testing.T) {
		versions, err := repo.ListTemplateVersions(ctx, "test-greeting")
		require.NoError(t, err)
		assert.Len(t, versions, 2)
		assert.Equal(t, 2, versions[0].Version)
		assert.Equal(t, 1, versions[1].Version)
	})

	// Collections test
	collection := &domain.PromptCollection{
		ID:          uuid.New(),
		Name:        "test-suite",
		Description: "A suite of prompt templates",
		Templates:   []uuid.UUID{pt1_v1.ID, pt2_v1.ID},
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
		collection.Templates = []uuid.UUID{pt1_v2.ID} // Update association to latest v2
		err := repo.UpdateCollection(ctx, collection)
		require.NoError(t, err)

		fetched, err := repo.GetCollectionByID(ctx, collection.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated test suite description", fetched.Description)
		assert.Equal(t, []uuid.UUID{pt1_v2.ID}, fetched.Templates)
	})

	t.Run("Resolve Collection Prompts (Only Active)", func(t *testing.T) {
		// We add pt2_v1 which is "draft" to the collection
		collection.Templates = []uuid.UUID{pt1_v1.ID, pt2_v1.ID}
		err := repo.UpdateCollection(ctx, collection)
		require.NoError(t, err)

		// Dynamic resolution:
		// pt1_v1 has name "test-greeting". Its latest *active* version is pt1_v2!
		// pt2_v1 has name "test-goodbye". Its latest version is pt2_v1 but its status is "draft" (not active), so it should not be resolved!
		resolved, err := repo.ResolveCollectionPrompts(ctx, collection.ID)
		require.NoError(t, err)
		
		// Should only resolve the active greeting prompt (pt1_v2)
		assert.Len(t, resolved, 1)
		assert.Equal(t, "test-greeting", resolved[0].Name)
		assert.Equal(t, 2, resolved[0].Version)
		assert.Equal(t, pt1_v2.ID, resolved[0].ID)
	})

	t.Run("Delete Collection", func(t *testing.T) {
		err := repo.DeleteCollection(ctx, collection.ID)
		require.NoError(t, err)

		_, err = repo.GetCollectionByID(ctx, collection.ID)
		assert.Error(t, err)
	})

	t.Run("Delete Template", func(t *testing.T) {
		err := repo.DeleteTemplate(ctx, pt1_v1.ID)
		require.NoError(t, err)

		_, err = repo.GetTemplateByID(ctx, pt1_v1.ID)
		assert.Error(t, err)
	})
}
