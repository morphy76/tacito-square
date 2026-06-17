package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// CommunityRepository implements the outbound.CommunityRepository port interface using PostgreSQL.
type CommunityRepository struct {
	pool *pgxpool.Pool
}

// NewCommunityRepository creates a new instance of CommunityRepository.
func NewCommunityRepository(pool *pgxpool.Pool) *CommunityRepository {
	return &CommunityRepository{pool: pool}
}

// Create inserts a new Community record into the database.
func (r *CommunityRepository) Create(ctx context.Context, c *model.Community) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	configJSON, err := json.Marshal(c.Configuration)
	if err != nil {
		return fmt.Errorf("marshal community config: %w", err)
	}

	query := `INSERT INTO communities (
		id, tenant_id, name, description, topology, configuration, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.pool.Exec(ctx, query,
		c.ID, c.TenantID, c.Name, c.Description, c.Topology, configJSON, c.Status, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create community: %w", err)
	}
	return nil
}

// GetByID retrieves a Community by its ID.
func (r *CommunityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Community, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, topology, configuration, status, created_at, updated_at
	FROM communities WHERE id = $1 AND tenant_id = $2`

	var c model.Community
	var configBytes []byte

	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Topology, &configBytes, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("community not found: %s", id)
		}
		return nil, fmt.Errorf("get community by id: %w", err)
	}

	if err := json.Unmarshal(configBytes, &c.Configuration); err != nil {
		return nil, fmt.Errorf("unmarshal community configuration: %w", err)
	}

	return &c, nil
}

// GetByName retrieves a Community by its name.
func (r *CommunityRepository) GetByName(ctx context.Context, name string) (*model.Community, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, topology, configuration, status, created_at, updated_at
	FROM communities WHERE name = $1 AND tenant_id = $2`

	var c model.Community
	var configBytes []byte

	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Topology, &configBytes, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("community not found by name: %s", name)
		}
		return nil, fmt.Errorf("get community by name: %w", err)
	}

	if err := json.Unmarshal(configBytes, &c.Configuration); err != nil {
		return nil, fmt.Errorf("unmarshal community configuration: %w", err)
	}

	return &c, nil
}

// List retrieves all Communities for the dynamic tenant context.
func (r *CommunityRepository) List(ctx context.Context) ([]*model.Community, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, topology, configuration, status, created_at, updated_at
	FROM communities WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list communities: %w", err)
	}
	defer rows.Close()

	var communities []*model.Community
	for rows.Next() {
		var c model.Community
		var configBytes []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Topology, &configBytes, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan community: %w", err)
		}

		if err := json.Unmarshal(configBytes, &c.Configuration); err != nil {
			return nil, fmt.Errorf("unmarshal community configuration: %w", err)
		}

		communities = append(communities, &c)
	}
	return communities, nil
}

// Update updates an existing Community record.
func (r *CommunityRepository) Update(ctx context.Context, c *model.Community) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	configJSON, err := json.Marshal(c.Configuration)
	if err != nil {
		return fmt.Errorf("marshal community config: %w", err)
	}

	// Fetch existing community topology to check if it's changing
	var existingTopology string
	err = r.pool.QueryRow(ctx, `SELECT topology FROM communities WHERE id = $1 AND tenant_id = $2`, c.ID, c.TenantID).Scan(&existingTopology)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("community not found: %s", c.ID)
		}
		return fmt.Errorf("check existing community topology: %w", err)
	}

	if existingTopology != string(c.Topology) {
		var count int
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE community_id = $1`, c.ID).Scan(&count)
		if err != nil {
			return fmt.Errorf("check community agents count for topology update: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("cannot change topology of a community with assigned agents")
		}
	}

	query := `UPDATE communities SET 
		name = $1, description = $2, topology = $3, configuration = $4, status = $5, updated_at = $6
	WHERE id = $7 AND tenant_id = $8`

	c.UpdatedAt = time.Now().UTC()
	cmdTag, err := r.pool.Exec(ctx, query,
		c.Name, c.Description, c.Topology, configJSON, c.Status, c.UpdatedAt, c.ID, c.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update community: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("community not found: %s", c.ID)
	}
	return nil
}

// Delete removes a Community from persistent storage.
func (r *CommunityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM communities WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete community: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("community not found: %s", id)
	}
	return nil
}
