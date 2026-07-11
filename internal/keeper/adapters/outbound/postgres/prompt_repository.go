package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// CreateTemplate inserts a new Prompt Template and initiates version 1.
func (r *PromptRepository) CreateTemplate(ctx context.Context, t *model.PromptTemplate) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	t.TenantID = ten.FullName()

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		query := `INSERT INTO prompt_templates (
			id, tenant_id, name, content, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := tx.Exec(ctx, query,
			t.ID, t.TenantID, t.Name, t.Content, t.Status, t.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create prompt template: %w", err)
		}

		// initiate version 1
		versionQuery := `INSERT INTO prompt_versions (
			id, prompt_id, version_number, content_snapshot, created_at
		) VALUES ($1, $2, $3, $4, $5)`
		versionID := uuid.New()
		_, err = tx.Exec(ctx, versionQuery,
			versionID, t.ID, 1, t.Content, t.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create prompt version 1: %w", err)
		}
		return nil
	})
}

// GetTemplateByID retrieves a specific template version by UUID, resolving the content from the latest version.
func (r *PromptRepository) GetTemplateByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT pt.id, pt.tenant_id, pt.name, 
		COALESCE((SELECT content_snapshot FROM prompt_versions WHERE prompt_id = pt.id ORDER BY version_number DESC LIMIT 1), pt.content), 
		pt.status, pt.created_at 
		FROM prompt_templates pt WHERE pt.id = $1 AND (pt.tenant_id = $2 OR pt.tenant_id = 'system')`
	var t model.PromptTemplate
	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Status, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("prompt template not found: %s", id)
		}
		return nil, fmt.Errorf("get prompt template by id: %w", err)
	}
	return &t, nil
}

// ListTemplates lists all templates for the tenant context, resolving the content from the latest version.
func (r *PromptRepository) ListTemplates(ctx context.Context) ([]*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT pt.id, pt.tenant_id, pt.name, 
		COALESCE((SELECT content_snapshot FROM prompt_versions WHERE prompt_id = pt.id ORDER BY version_number DESC LIMIT 1), pt.content), 
		pt.status, pt.created_at 
		FROM prompt_templates pt 
		WHERE pt.tenant_id = $1
		ORDER BY pt.name ASC`
	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []*model.PromptTemplate
	for rows.Next() {
		var t model.PromptTemplate
		err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, &t)
	}
	if templates == nil {
		templates = make([]*model.PromptTemplate, 0)
	}
	return templates, nil
}

// UpdateTemplate updates an existing Prompt Template. If the content is modified, a new version is created in prompt_versions.
func (r *PromptRepository) UpdateTemplate(ctx context.Context, t *model.PromptTemplate) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	t.TenantID = ten.FullName()

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// 1. Fetch current latest content and version number
		var currentContent string
		var currentVerNum int
		err := tx.QueryRow(ctx, `
			SELECT 
				COALESCE((SELECT content_snapshot FROM prompt_versions WHERE prompt_id = $1 ORDER BY version_number DESC LIMIT 1), content),
				COALESCE((SELECT version_number FROM prompt_versions WHERE prompt_id = $1 ORDER BY version_number DESC LIMIT 1), 0)
			FROM prompt_templates WHERE id = $1 AND tenant_id = $2`, t.ID, t.TenantID).Scan(&currentContent, &currentVerNum)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("prompt template not found: %s", t.ID)
			}
			return fmt.Errorf("fetch prompt template current content: %w", err)
		}

		// 2. Update prompt template metadata (name, status) but NOT content
		query := `UPDATE prompt_templates SET name = $1, status = $2 WHERE id = $3 AND tenant_id = $4`
		cmdTag, err := tx.Exec(ctx, query, t.Name, t.Status, t.ID, t.TenantID)
		if err != nil {
			return fmt.Errorf("update prompt template: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return fmt.Errorf("prompt template not found: %s", t.ID)
		}

		// 3. If content has changed, create a new version
		if t.Content != currentContent {
			newVerNum := currentVerNum + 1
			versionID := uuid.New()
			now := time.Now().UTC()
			_, err = tx.Exec(ctx, `INSERT INTO prompt_versions (id, prompt_id, version_number, content_snapshot, created_at) VALUES ($1, $2, $3, $4, $5)`,
				versionID, t.ID, newVerNum, t.Content, now)
			if err != nil {
				return fmt.Errorf("create new prompt version: %w", err)
			}
		}
		return nil
	})
}

// DeleteTemplate deletes a specific template.
func (r *PromptRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM prompt_templates WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("prompt template not found: %s", id)
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
	if collections == nil {
		collections = make([]*model.PromptCollection, 0)
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
	cmdTag, err := tx.Exec(ctx, query, c.Name, c.Description, c.UpdatedAt, c.ID, c.TenantID)
	if err != nil {
		return fmt.Errorf("update prompt collection: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("prompt collection not found: %s", c.ID)
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
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("prompt collection not found: %s", id)
	}
	return nil
}

// ResolveCollectionPrompts resolves the active templates associated with the collection, content fetched from latest version.
func (r *PromptRepository) ResolveCollectionPrompts(ctx context.Context, collectionID uuid.UUID) ([]*model.PromptTemplate, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT pt.id, pt.tenant_id, pt.name, 
		COALESCE((SELECT content_snapshot FROM prompt_versions WHERE prompt_id = pt.id ORDER BY version_number DESC LIMIT 1), pt.content), 
		pt.status, pt.created_at
		FROM prompt_templates pt
		JOIN prompt_collection_templates pct ON pt.id = pct.prompt_template_id
		WHERE pct.prompt_collection_id = $1 AND (pt.tenant_id = $2 OR pt.tenant_id = 'system') AND pt.status = $3
		ORDER BY pt.name ASC`
	rows, err := r.pool.Query(ctx, query, collectionID, ten.FullName(), model.PromptStatusActive)
	if err != nil {
		return nil, fmt.Errorf("resolve collection prompts: %w", err)
	}
	defer rows.Close()

	var resolved []*model.PromptTemplate
	for rows.Next() {
		var t model.PromptTemplate
		err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan resolved template: %w", err)
		}
		resolved = append(resolved, &t)
	}
	if resolved == nil {
		resolved = make([]*model.PromptTemplate, 0)
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

// CreateVersion inserts a new PromptVersion.
func (r *PromptRepository) CreateVersion(ctx context.Context, v *model.PromptVersion) error {
	query := `INSERT INTO prompt_versions (id, prompt_id, version_number, content_snapshot, created_at) 
		VALUES ($1, $2, $3, $4, $5)`
	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, v.ID, v.PromptID, v.VersionNumber, v.ContentSnapshot, v.CreatedAt)
	if err != nil {
		return fmt.Errorf("create prompt version: %w", err)
	}
	return nil
}

// GetLatestVersion retrieves the latest version record for a prompt template.
func (r *PromptRepository) GetLatestVersion(ctx context.Context, promptID uuid.UUID) (*model.PromptVersion, error) {
	query := `SELECT id, prompt_id, version_number, content_snapshot, created_at 
		FROM prompt_versions WHERE prompt_id = $1 ORDER BY version_number DESC LIMIT 1`
	exec := GetExecutor(ctx, r.pool)
	var v model.PromptVersion
	err := exec.QueryRow(ctx, query, promptID).Scan(&v.ID, &v.PromptID, &v.VersionNumber, &v.ContentSnapshot, &v.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("no versions found for prompt: %s", promptID)
		}
		return nil, fmt.Errorf("get latest prompt version: %w", err)
	}
	return &v, nil
}

// AddPromptToCollection adds a prompt template to a prompt collection. Returns 409 conflict if duplicate.
func (r *PromptRepository) AddPromptToCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	query := `INSERT INTO prompt_collection_templates (prompt_collection_id, prompt_template_id) VALUES ($1, $2)`
	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, collectionID, promptID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("prompt is already in collection: 409 Conflict")
		}
		return fmt.Errorf("add prompt to collection: %w", err)
	}
	return nil
}

// RemovePromptFromCollection removes a prompt template from a prompt collection.
func (r *PromptRepository) RemovePromptFromCollection(ctx context.Context, collectionID uuid.UUID, promptID uuid.UUID) error {
	query := `DELETE FROM prompt_collection_templates WHERE prompt_collection_id = $1 AND prompt_template_id = $2`
	exec := GetExecutor(ctx, r.pool)
	cmdTag, err := exec.Exec(ctx, query, collectionID, promptID)
	if err != nil {
		return fmt.Errorf("remove prompt from collection: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("prompt template reference not found in collection: %s", promptID)
	}
	return nil
}
