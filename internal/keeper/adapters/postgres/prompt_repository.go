package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
)

// PromptRepository implements outbound.PromptRepository port using pgxpool.
type PromptRepository struct {
	pool *pgxpool.Pool
}

// NewPromptRepository creates a new PromptRepository.
func NewPromptRepository(pool *pgxpool.Pool) *PromptRepository {
	return &PromptRepository{pool: pool}
}

// CreateTemplate inserts a new Prompt Template revision.
func (r *PromptRepository) CreateTemplate(ctx context.Context, t *domain.PromptTemplate) error {
	query := `INSERT INTO prompt_templates (
		id, name, content, role, version, status, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query,
		t.ID, t.Name, t.Content, t.Role, t.Version, t.Status, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create prompt template: %w", err)
	}
	return nil
}

// GetTemplateByID retrieves a specific template version by UUID.
func (r *PromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*domain.PromptTemplate, error) {
	query := `SELECT id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE id = $1`
	var t domain.PromptTemplate
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("prompt template not found: %s", id)
		}
		return nil, fmt.Errorf("get prompt template by id: %w", err)
	}
	return &t, nil
}

// GetLatestTemplateByName retrieves the highest version of a template name.
func (r *PromptRepository) GetLatestTemplateByName(ctx context.Context, name string) (*domain.PromptTemplate, error) {
	query := `SELECT id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE name = $1 ORDER BY version DESC LIMIT 1`
	var t domain.PromptTemplate
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&t.ID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("prompt template not found by name: %s", name)
		}
		return nil, fmt.Errorf("get latest prompt template by name: %w", err)
	}
	return &t, nil
}

// ListTemplates lists the latest versions of unique template names.
func (r *PromptRepository) ListTemplates(ctx context.Context) ([]*domain.PromptTemplate, error) {
	query := `SELECT DISTINCT ON (name) id, name, content, role, version, status, created_at 
		FROM prompt_templates 
		ORDER BY name, version DESC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []*domain.PromptTemplate
	for rows.Next() {
		var t domain.PromptTemplate
		err := rows.Scan(&t.ID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

// ListTemplateVersions lists all historical versions of a template name.
func (r *PromptRepository) ListTemplateVersions(ctx context.Context, name string) ([]*domain.PromptTemplate, error) {
	query := `SELECT id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE name = $1 ORDER BY version DESC`
	rows, err := r.pool.Query(ctx, query, name)
	if err != nil {
		return nil, fmt.Errorf("list template versions: %w", err)
	}
	defer rows.Close()

	var templates []*domain.PromptTemplate
	for rows.Next() {
		var t domain.PromptTemplate
		err := rows.Scan(&t.ID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template version: %w", err)
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

// DeleteTemplate deletes a specific template version.
func (r *PromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM prompt_templates WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// CreateCollection saves a prompt collection and its linked template UUID associations.
func (r *PromptRepository) CreateCollection(ctx context.Context, c *domain.PromptCollection) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO prompt_collections (
		id, name, description, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(ctx, query, c.ID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert prompt collection: %w", err)
	}

	if err := r.saveCollectionTemplates(ctx, tx, c.ID, c.Templates); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetCollectionByID loads collection details and linked template UUIDs.
func (r *PromptRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*domain.PromptCollection, error) {
	query := `SELECT id, name, description, created_at, updated_at 
		FROM prompt_collections WHERE id = $1`
	var c domain.PromptCollection
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("prompt collection not found: %s", id)
		}
		return nil, fmt.Errorf("get prompt collection: %w", err)
	}

	templates, err := r.loadCollectionTemplates(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Templates = templates

	return &c, nil
}

// ListCollections lists all collections.
func (r *PromptRepository) ListCollections(ctx context.Context) ([]*domain.PromptCollection, error) {
	query := `SELECT id, name, description, created_at, updated_at 
		FROM prompt_collections ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	var collections []*domain.PromptCollection
	for rows.Next() {
		var c domain.PromptCollection
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		templates, err := r.loadCollectionTemplates(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Templates = templates
		collections = append(collections, &c)
	}
	return collections, nil
}

// UpdateCollection updates a collection and refreshes its templates list.
func (r *PromptRepository) UpdateCollection(ctx context.Context, c *domain.PromptCollection) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	c.UpdatedAt = time.Now().UTC()
	query := `UPDATE prompt_collections SET 
		name = $1, description = $2, updated_at = $3 
		WHERE id = $4`
	_, err = tx.Exec(ctx, query, c.Name, c.Description, c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("update prompt collection: %w", err)
	}

	if err := r.saveCollectionTemplates(ctx, tx, c.ID, c.Templates); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// DeleteCollection deletes a collection.
func (r *PromptRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM prompt_collections WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}

// ResolveCollectionPrompts resolves the latest active version of each template name associated with the collection.
func (r *PromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*domain.PromptTemplate, error) {
	query := `SELECT DISTINCT ON (pt.name) pt.id, pt.name, pt.content, pt.role, pt.version, pt.status, pt.created_at
		FROM prompt_templates pt
		WHERE pt.name IN (
			SELECT name FROM prompt_templates 
			JOIN prompt_collection_templates pct ON prompt_templates.id = pct.prompt_template_id
			WHERE pct.prompt_collection_id = $1
		)
		AND pt.status = $2
		ORDER BY pt.name, pt.version DESC`
	rows, err := r.pool.Query(ctx, query, collectionID, domain.PromptStatusActive)
	if err != nil {
		return nil, fmt.Errorf("resolve collection prompts: %w", err)
	}
	defer rows.Close()

	var resolved []*domain.PromptTemplate
	for rows.Next() {
		var t domain.PromptTemplate
		err := rows.Scan(&t.ID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan resolved template: %w", err)
		}
		resolved = append(resolved, &t)
	}
	return resolved, nil
}

// --- Helper methods ---

func (r *PromptRepository) loadCollectionTemplates(ctx context.Context, collectionID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT prompt_template_id FROM prompt_collection_templates WHERE prompt_collection_id = $1`
	rows, err := r.pool.Query(ctx, query, collectionID)
	if err != nil {
		return nil, fmt.Errorf("load collection templates: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan collection template id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PromptRepository) saveCollectionTemplates(ctx context.Context, tx pgx.Tx, collectionID uuid.UUID, templates []uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM prompt_collection_templates WHERE prompt_collection_id = $1`, collectionID)
	if err != nil {
		return fmt.Errorf("delete collection templates: %w", err)
	}

	for _, tID := range templates {
		_, err := tx.Exec(ctx, `INSERT INTO prompt_collection_templates (prompt_collection_id, prompt_template_id) VALUES ($1, $2)`, collectionID, tID)
		if err != nil {
			return fmt.Errorf("insert collection template: %w", err)
		}
	}
	return nil
}
