package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
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
func (r *PromptRepository) CreateTemplate(ctx context.Context, t *model.PromptTemplate) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	t.TenantID = ten.FullName()

	query := `INSERT INTO prompt_templates (
		id, tenant_id, name, content, role, version, status, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		t.ID, t.TenantID, t.Name, t.Content, t.Role, t.Version, t.Status, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create prompt template: %w", err)
	}
	return nil
}

// GetTemplateByID retrieves a specific template version by UUID.
func (r *PromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE id = $1 AND tenant_id = $2`
	var t model.PromptTemplate
	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt,
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
func (r *PromptRepository) GetLatestTemplateByName(ctx context.Context, name string) (*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE name = $1 AND tenant_id = $2 ORDER BY version DESC LIMIT 1`
	var t model.PromptTemplate
	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt,
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
func (r *PromptRepository) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT DISTINCT ON (name) id, tenant_id, name, content, role, version, status, created_at 
		FROM prompt_templates 
		WHERE tenant_id = $1
		ORDER BY name, version DESC`
	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []*model.PromptTemplate
	for rows.Next() {
		var t model.PromptTemplate
		err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

// ListTemplateVersions lists all historical versions of a template name.
func (r *PromptRepository) ListTemplateVersions(ctx context.Context, name string) ([]*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, content, role, version, status, created_at 
		FROM prompt_templates WHERE name = $1 AND tenant_id = $2 ORDER BY version DESC`
	rows, err := r.pool.Query(ctx, query, name, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list template versions: %w", err)
	}
	defer rows.Close()

	var templates []*model.PromptTemplate
	for rows.Next() {
		var t model.PromptTemplate
		err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template version: %w", err)
		}
		templates = append(templates, &t)
	}
	return templates, nil
}

// DeleteTemplate deletes a specific template version.
func (r *PromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM prompt_templates WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// CreateCollection saves a prompt collection and its linked template UUID associations.
func (r *PromptRepository) CreateCollection(ctx context.Context, c *model.PromptCollection) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO prompt_collections (
		id, tenant_id, name, description, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(ctx, query, c.ID, c.TenantID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt)
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
func (r *PromptRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.PromptCollection, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, created_at, updated_at 
		FROM prompt_collections WHERE id = $1 AND tenant_id = $2`
	var c model.PromptCollection
	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
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
func (r *PromptRepository) ListCollections(ctx context.Context) ([]*model.PromptCollection, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, created_at, updated_at 
		FROM prompt_collections WHERE tenant_id = $1 ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	var collections []*model.PromptCollection
	for rows.Next() {
		var c model.PromptCollection
		err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
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
func (r *PromptRepository) UpdateCollection(ctx context.Context, c *model.PromptCollection) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	c.UpdatedAt = time.Now().UTC()
	query := `UPDATE prompt_collections SET 
		name = $1, description = $2, updated_at = $3 
		WHERE id = $4 AND tenant_id = $5`
	_, err = tx.Exec(ctx, query, c.Name, c.Description, c.UpdatedAt, c.ID, c.TenantID)
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
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM prompt_collections WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}

// ResolveCollectionPrompts resolves the latest active version of each template name associated with the collection.
func (r *PromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT DISTINCT ON (pt.name) pt.id, pt.tenant_id, pt.name, pt.content, pt.role, pt.version, pt.status, pt.created_at
		FROM prompt_templates pt
		WHERE pt.name IN (
			SELECT name FROM prompt_templates 
			JOIN prompt_collection_templates pct ON prompt_templates.id = pct.prompt_template_id
			WHERE pct.prompt_collection_id = $1 AND prompt_templates.tenant_id = $2
		)
		AND pt.status = $3
		AND pt.tenant_id = $2
		ORDER BY pt.name, pt.version DESC`
	rows, err := r.pool.Query(ctx, query, collectionID, ten.FullName(), model.PromptStatusActive)
	if err != nil {
		return nil, fmt.Errorf("resolve collection prompts: %w", err)
	}
	defer rows.Close()

	var resolved []*model.PromptTemplate
	for rows.Next() {
		var t model.PromptTemplate
		err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Role, &t.Version, &t.Status, &t.CreatedAt)
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
