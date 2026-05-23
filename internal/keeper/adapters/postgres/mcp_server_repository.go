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
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// MCPServerRepository implements the outbound.MCPServerRepository port interface using PostgreSQL.
type MCPServerRepository struct {
	pool *pgxpool.Pool
}

// NewMCPServerRepository creates a new instance of MCPServerRepository.
func NewMCPServerRepository(pool *pgxpool.Pool) *MCPServerRepository {
	return &MCPServerRepository{pool: pool}
}

// Create inserts a new MCP server configuration into the database.
func (r *MCPServerRepository) Create(ctx context.Context, s *domain.MCPServer) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	argsJSON, err := json.Marshal(s.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	envJSON, err := json.Marshal(s.Env)
	if err != nil {
		return fmt.Errorf("marshal env: %w", err)
	}

	query := `INSERT INTO mcp_servers (
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err = r.pool.Exec(ctx, query,
		s.ID, s.TenantID, s.Name, s.Description, s.Transport, s.Command, argsJSON, envJSON,
		s.URL, s.AuthSecretRef, s.Status, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create mcp server: %w", err)
	}
	return nil
}

// GetByID retrieves an MCP server configuration by its ID.
func (r *MCPServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MCPServer, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_servers WHERE id = $1 AND tenant_id = $2`

	var s domain.MCPServer
	var argsBytes []byte
	var envBytes []byte

	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Transport, &s.Command, &argsBytes, &envBytes,
		&s.URL, &s.AuthSecretRef, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mcp server not found: %s", id)
		}
		return nil, fmt.Errorf("get mcp server by id: %w", err)
	}

	if len(argsBytes) > 0 {
		if err := json.Unmarshal(argsBytes, &s.Args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
	}
	if len(envBytes) > 0 {
		if err := json.Unmarshal(envBytes, &s.Env); err != nil {
			return nil, fmt.Errorf("unmarshal env: %w", err)
		}
	}

	return &s, nil
}

// GetByName retrieves an MCP server configuration by its unique name.
func (r *MCPServerRepository) GetByName(ctx context.Context, name string) (*domain.MCPServer, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_servers WHERE name = $1 AND tenant_id = $2`

	var s domain.MCPServer
	var argsBytes []byte
	var envBytes []byte

	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Transport, &s.Command, &argsBytes, &envBytes,
		&s.URL, &s.AuthSecretRef, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("mcp server not found by name: %s", name)
		}
		return nil, fmt.Errorf("get mcp server by name: %w", err)
	}

	if len(argsBytes) > 0 {
		if err := json.Unmarshal(argsBytes, &s.Args); err != nil {
			return nil, fmt.Errorf("unmarshal args: %w", err)
		}
	}
	if len(envBytes) > 0 {
		if err := json.Unmarshal(envBytes, &s.Env); err != nil {
			return nil, fmt.Errorf("unmarshal env: %w", err)
		}
	}

	return &s, nil
}

// List retrieves all MCP server configurations.
func (r *MCPServerRepository) List(ctx context.Context) ([]*domain.MCPServer, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, transport, command, args, env, 
		url, auth_secret_ref, status, created_at, updated_at
	FROM mcp_servers WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list mcp servers: %w", err)
	}
	defer rows.Close()

	var servers []*domain.MCPServer
	for rows.Next() {
		var s domain.MCPServer
		var argsBytes []byte
		var envBytes []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Transport, &s.Command, &argsBytes, &envBytes,
			&s.URL, &s.AuthSecretRef, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mcp server: %w", err)
		}

		if len(argsBytes) > 0 {
			if err := json.Unmarshal(argsBytes, &s.Args); err != nil {
				return nil, fmt.Errorf("unmarshal args: %w", err)
			}
		}
		if len(envBytes) > 0 {
			if err := json.Unmarshal(envBytes, &s.Env); err != nil {
				return nil, fmt.Errorf("unmarshal env: %w", err)
			}
		}

		servers = append(servers, &s)
	}
	return servers, nil
}

// Update updates an existing MCP server configuration.
func (r *MCPServerRepository) Update(ctx context.Context, s *domain.MCPServer) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	argsJSON, err := json.Marshal(s.Args)
	if err != nil {
		return fmt.Errorf("marshal args: %w", err)
	}
	envJSON, err := json.Marshal(s.Env)
	if err != nil {
		return fmt.Errorf("marshal env: %w", err)
	}

	query := `UPDATE mcp_servers SET 
		name = $1, description = $2, transport = $3, command = $4, args = $5, env = $6, 
		url = $7, auth_secret_ref = $8, status = $9, updated_at = $10
	WHERE id = $11 AND tenant_id = $12`

	s.UpdatedAt = time.Now().UTC()
	_, err = r.pool.Exec(ctx, query,
		s.Name, s.Description, s.Transport, s.Command, argsJSON, envJSON,
		s.URL, s.AuthSecretRef, s.Status, s.UpdatedAt, s.ID, s.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update mcp server: %w", err)
	}
	return nil
}

// Delete removes an MCP server configuration from the database by its ID.
func (r *MCPServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM mcp_servers WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete mcp server: %w", err)
	}
	return nil
}
