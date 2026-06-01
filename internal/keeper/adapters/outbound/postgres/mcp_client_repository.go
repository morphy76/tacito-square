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

// MCPClientRepository implements the outbound.MCPClientRepository port interface using PostgreSQL.
type MCPClientRepository struct {
	pool *pgxpool.Pool
}

// NewMCPClientRepository creates a new instance of MCPClientRepository.
func NewMCPClientRepository(pool *pgxpool.Pool) *MCPClientRepository {
	return &MCPClientRepository{pool: pool}
}

// Create inserts a new MCP client configuration into the database.
func (r *MCPClientRepository) Create(ctx context.Context, c *model.MCPClient) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	argsJSON, err := json.Marshal(c.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	envJSON, err := json.Marshal(c.Env)
	if err != nil {
		return fmt.Errorf("marshal env: %w", err)
	}

	query := `INSERT INTO mcp_clients (
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = r.pool.Exec(ctx, query,
		c.ID, c.TenantID, c.Name, c.Description, c.Transport, c.Command, argsJSON, envJSON,
		c.URL, c.AuthSecretRef, c.Status, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create mcp client: %w", err)
	}
	return nil
}

// GetByID retrieves an MCP client configuration by its ID.
func (r *MCPClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MCPClient, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_clients WHERE id = $1 AND tenant_id = $2`

	var c model.MCPClient
	var argsBytes []byte
	var envBytes []byte

	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Transport, &c.Command, &argsBytes, &envBytes,
		&c.URL, &c.AuthSecretRef, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mcp client not found: %s", id)
		}
		return nil, fmt.Errorf("get mcp client by id: %w", err)
	}

	if len(argsBytes) > 0 {
		if err := json.Unmarshal(argsBytes, &c.Args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
	}
	if len(envBytes) > 0 {
		if err := json.Unmarshal(envBytes, &c.Env); err != nil {
			return nil, fmt.Errorf("unmarshal env: %w", err)
		}
	}

	return &c, nil
}

// GetByName retrieves an MCP client configuration by its unique name.
func (r *MCPClientRepository) GetByName(ctx context.Context, name string) (*model.MCPClient, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_clients WHERE name = $1 AND tenant_id = $2`

	var c model.MCPClient
	var argsBytes []byte
	var envBytes []byte

	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Transport, &c.Command, &argsBytes, &envBytes,
		&c.URL, &c.AuthSecretRef, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mcp client not found by name: %s", name)
		}
		return nil, fmt.Errorf("get mcp client by name: %w", err)
	}

	if len(argsBytes) > 0 {
		if err := json.Unmarshal(argsBytes, &c.Args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
	}
	if len(envBytes) > 0 {
		if err := json.Unmarshal(envBytes, &c.Env); err != nil {
			return nil, fmt.Errorf("unmarshal env: %w", err)
		}
	}

	return &c, nil
}

// List retrieves all MCP client configurations.
func (r *MCPClientRepository) List(ctx context.Context) ([]*model.MCPClient, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_clients WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list mcp clients: %w", err)
	}
	defer rows.Close()

	var clients []*model.MCPClient
	for rows.Next() {
		var c model.MCPClient
		var argsBytes []byte
		var envBytes []byte

		err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Transport, &c.Command, &argsBytes, &envBytes,
			&c.URL, &c.AuthSecretRef, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mcp client: %w", err)
		}

		if len(argsBytes) > 0 {
			if err := json.Unmarshal(argsBytes, &c.Args); err != nil {
				return nil, fmt.Errorf("unmarshal args: %w", err)
			}
		}
		if len(envBytes) > 0 {
			if err := json.Unmarshal(envBytes, &c.Env); err != nil {
				return nil, fmt.Errorf("unmarshal env: %w", err)
			}
		}

		clients = append(clients, &c)
	}
	return clients, nil
}

// Update updates an existing MCP client configuration.
func (r *MCPClientRepository) Update(ctx context.Context, c *model.MCPClient) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	c.TenantID = ten.FullName()

	argsJSON, err := json.Marshal(c.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	envJSON, err := json.Marshal(c.Env)
	if err != nil {
		return fmt.Errorf("marshal env: %w", err)
	}

	query := `UPDATE mcp_clients SET 
		name = $1, description = $2, transport = $3, command = $4, args = $5, env = $6, 
		url = $7, auth_secret_ref = $8, status = $9, updated_at = $10
	WHERE id = $11 AND tenant_id = $12`

	c.UpdatedAt = time.Now().UTC()
	cmdTag, err := r.pool.Exec(ctx, query,
		c.Name, c.Description, c.Transport, c.Command, argsJSON, envJSON,
		c.URL, c.AuthSecretRef, c.Status, c.UpdatedAt, c.ID, c.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update mcp client: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("mcp client not found: %s", c.ID)
	}
	return nil
}

// Delete removes an MCP client configuration from the database by its ID.
func (r *MCPClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM mcp_clients WHERE id = $1 AND tenant_id = $2`
	cmdTag, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete mcp client: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("mcp client not found: %s", id)
	}
	return nil
}
