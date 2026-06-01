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

// SkillRepository implements the outbound.SkillRepository port interface using PostgreSQL.
type SkillRepository struct {
	pool *pgxpool.Pool
}

// NewSkillRepository creates a new instance of SkillRepository.
func NewSkillRepository(pool *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{pool: pool}
}

// Create inserts a new Skill capability into the database.
func (r *SkillRepository) Create(ctx context.Context, s *model.Skill) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	query := `INSERT INTO skills (
		id, tenant_id, name, description, content, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		s.ID, s.TenantID, s.Name, s.Description, s.Content, s.Status, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert skill: %w", err)
	}
	return nil
}

// GetByID retrieves a Skill by its ID.
func (r *SkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, content, status, created_at, updated_at
		FROM skills WHERE id = $1 AND tenant_id = $2`

	var s model.Skill
	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Content, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("skill not found: %s", id)
		}
		return nil, fmt.Errorf("get skill by id: %w", err)
	}

	return &s, nil
}

// GetByName retrieves a Skill by its unique name.
func (r *SkillRepository) GetByName(ctx context.Context, name string) (*model.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, content, status, created_at, updated_at
		FROM skills WHERE name = $1 AND tenant_id = $2`

	var s model.Skill
	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Content, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("skill not found by name: %s", name)
		}
		return nil, fmt.Errorf("get skill by name: %w", err)
	}

	return &s, nil
}

// List retrieves all Skills.
func (r *SkillRepository) List(ctx context.Context) ([]*model.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, content, status, created_at, updated_at
		FROM skills WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	var skills []*model.Skill
	for rows.Next() {
		var s model.Skill
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Content, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		skills = append(skills, &s)
	}
	if skills == nil {
		skills = make([]*model.Skill, 0)
	}
	return skills, nil
}

// Update updates an existing Skill properties.
func (r *SkillRepository) Update(ctx context.Context, s *model.Skill) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	query := `UPDATE skills SET name = $1, description = $2, content = $3, status = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7`

	s.UpdatedAt = time.Now().UTC()
	cmdTag, err := r.pool.Exec(ctx, query,
		s.Name, s.Description, s.Content, s.Status, s.UpdatedAt, s.ID, s.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found: %s", s.ID)
	}
	return nil
}

// Delete removes a Skill from the database.
func (r *SkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM skills WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found: %s", id)
	}
	return nil
}

// AttachSkillToAgent associates a skill with an agent.
func (r *SkillRepository) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `INSERT INTO agent_skills (agent_id, skill_id)
		SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM skills WHERE id = $2 AND tenant_id = $3)
		ON CONFLICT (agent_id, skill_id) DO NOTHING`
	cmdTag, err := r.pool.Exec(ctx, query, agentID, skillID, ten.FullName())
	if err != nil {
		return fmt.Errorf("attach skill to agent: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found: %s", skillID)
	}
	return nil
}

// DetachSkillFromAgent removes the association between a skill and an agent.
func (r *SkillRepository) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM agent_skills WHERE agent_id = $1 AND skill_id = $2 AND EXISTS (SELECT 1 FROM skills WHERE id = $2 AND tenant_id = $3)`
	cmdTag, err := r.pool.Exec(ctx, query, agentID, skillID, ten.FullName())
	if err != nil {
		return fmt.Errorf("detach skill from agent: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found: %s", skillID)
	}
	return nil
}

// ListSkillsByAgent retrieves all skills assigned to an agent.
func (r *SkillRepository) ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*model.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT s.id, s.tenant_id, s.name, s.description, s.content, s.status, s.created_at, s.updated_at
		FROM skills s
		JOIN agent_skills ags ON s.id = ags.skill_id
		WHERE ags.agent_id = $1 AND s.tenant_id = $2
		ORDER BY s.name ASC`

	rows, err := r.pool.Query(ctx, query, agentID, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list skills by agent: %w", err)
	}
	defer rows.Close()

	var skills []*model.Skill
	for rows.Next() {
		var s model.Skill
		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Content, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skill by agent: %w", err)
		}
		skills = append(skills, &s)
	}
	if skills == nil {
		skills = make([]*model.Skill, 0)
	}
	return skills, nil
}

// CreateCollection saves a skill collection.
func (r *SkillRepository) CreateCollection(ctx context.Context, c *model.SkillCollection) error {
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

	query := `INSERT INTO skill_collections (
		id, tenant_id, name, description, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(ctx, query, c.ID, c.TenantID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert skill collection: %w", err)
	}

	if err := r.saveCollectionSkills(ctx, tx, c.ID, c.Skills); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetCollectionByID loads collection details and linked skill UUIDs.
func (r *SkillRepository) GetCollectionByID(ctx context.Context, id uuid.UUID) (*model.SkillCollection, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, created_at, updated_at 
		FROM skill_collections WHERE id = $1 AND tenant_id = $2`
	var c model.SkillCollection
	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("skill collection not found: %s", id)
		}
		return nil, fmt.Errorf("get skill collection: %w", err)
	}

	skills, err := r.loadCollectionSkills(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Skills = skills

	return &c, nil
}

// ListCollections lists all collections.
func (r *SkillRepository) ListCollections(ctx context.Context) ([]*model.SkillCollection, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT id, tenant_id, name, description, created_at, updated_at 
		FROM skill_collections WHERE tenant_id = $1 ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer rows.Close()

	var collections []*model.SkillCollection
	for rows.Next() {
		var c model.SkillCollection
		err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan collection: %w", err)
		}
		skills, err := r.loadCollectionSkills(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		c.Skills = skills
		collections = append(collections, &c)
	}
	if collections == nil {
		collections = make([]*model.SkillCollection, 0)
	}
	return collections, nil
}

// UpdateCollection updates a collection and its skills.
func (r *SkillRepository) UpdateCollection(ctx context.Context, c *model.SkillCollection) error {
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
	query := `UPDATE skill_collections SET name = $1, description = $2, updated_at = $3 
		WHERE id = $4 AND tenant_id = $5`
	cmdTag, err := tx.Exec(ctx, query, c.Name, c.Description, c.UpdatedAt, c.ID, c.TenantID)
	if err != nil {
		return fmt.Errorf("update skill collection: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill collection not found: %s", c.ID)
	}

	if err := r.saveCollectionSkills(ctx, tx, c.ID, c.Skills); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// DeleteCollection deletes a collection.
func (r *SkillRepository) DeleteCollection(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM skill_collections WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete skill collection: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("skill collection not found: %s", id)
	}
	return nil
}

// ResolveCollectionSkills resolves active skills inside the collection.
func (r *SkillRepository) ResolveCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]*model.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT s.id, s.tenant_id, s.name, s.description, s.content, s.status, s.created_at, s.updated_at
		FROM skills s
		JOIN skill_collection_skills scs ON s.id = scs.skill_id
		WHERE scs.skill_collection_id = $1 AND s.tenant_id = $2 AND s.status = $3
		ORDER BY s.name ASC`
	rows, err := r.pool.Query(ctx, query, collectionID, ten.FullName(), model.SkillStatusActive)
	if err != nil {
		return nil, fmt.Errorf("resolve collection skills: %w", err)
	}
	defer rows.Close()

	var resolved []*model.Skill
	for rows.Next() {
		var s model.Skill
		err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Content, &s.Status, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan resolved skill: %w", err)
		}
		resolved = append(resolved, &s)
	}
	if resolved == nil {
		resolved = make([]*model.Skill, 0)
	}
	return resolved, nil
}

// --- Relational helper methods ---

func (r *SkillRepository) loadCollectionSkills(ctx context.Context, collectionID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT skill_id FROM skill_collection_skills WHERE skill_collection_id = $1`
	rows, err := r.pool.Query(ctx, query, collectionID)
	if err != nil {
		return nil, fmt.Errorf("load collection skills: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan collection skill id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *SkillRepository) saveCollectionSkills(ctx context.Context, tx pgx.Tx, collectionID uuid.UUID, skills []uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM skill_collection_skills WHERE skill_collection_id = $1`, collectionID)
	if err != nil {
		return fmt.Errorf("delete collection skills: %w", err)
	}

	for _, sID := range skills {
		_, err = tx.Exec(ctx, `INSERT INTO skill_collection_skills (skill_collection_id, skill_id) VALUES ($1, $2)`, collectionID, sID)
		if err != nil {
			return fmt.Errorf("insert collection skill: %w", err)
		}
	}
	return nil
}
