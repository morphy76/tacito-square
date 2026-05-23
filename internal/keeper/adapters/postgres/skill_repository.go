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

// SkillRepository implements the outbound.SkillRepository port interface using PostgreSQL.
type SkillRepository struct {
	pool *pgxpool.Pool
}

// NewSkillRepository creates a new instance of SkillRepository.
func NewSkillRepository(pool *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{pool: pool}
}

// Create inserts a new Skill Collection and its MCP server associations into the database.
func (r *SkillRepository) Create(ctx context.Context, s *domain.Skill) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	allowedJSON, err := json.Marshal(s.AllowedTools)
	if err != nil {
		return fmt.Errorf("marshal allowed tools: %w", err)
	}
	deniedJSON, err := json.Marshal(s.DeniedTools)
	if err != nil {
		return fmt.Errorf("marshal denied tools: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO skills (
		id, tenant_id, name, description, allowed_tools, denied_tools, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = tx.Exec(ctx, query,
		s.ID, s.TenantID, s.Name, s.Description, allowedJSON, deniedJSON, s.Status, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert skill: %w", err)
	}

	if err := r.saveMCPServers(ctx, tx, s.ID, s.MCPServers); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetByID retrieves a Skill Collection by its ID, loading its associated MCP server UUIDs.
func (r *SkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, allowed_tools, denied_tools, status, created_at, updated_at
	FROM skills WHERE id = $1 AND tenant_id = $2`

	var s domain.Skill
	var allowedBytes []byte
	var deniedBytes []byte

	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &allowedBytes, &deniedBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("skill not found: %s", id)
		}
		return nil, fmt.Errorf("get skill by id: %w", err)
	}

	if len(allowedBytes) > 0 {
		if err := json.Unmarshal(allowedBytes, &s.AllowedTools); err != nil {
			return nil, fmt.Errorf("unmarshal allowed tools: %w", err)
		}
	}
	if len(deniedBytes) > 0 {
		if err := json.Unmarshal(deniedBytes, &s.DeniedTools); err != nil {
			return nil, fmt.Errorf("unmarshal denied tools: %w", err)
		}
	}

	servers, err := r.loadMCPServers(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("load mcp servers for skill: %w", err)
	}
	s.MCPServers = servers

	return &s, nil
}

// GetByName retrieves a Skill Collection by its unique name.
func (r *SkillRepository) GetByName(ctx context.Context, name string) (*domain.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, allowed_tools, denied_tools, status, created_at, updated_at
	FROM skills WHERE name = $1 AND tenant_id = $2`

	var s domain.Skill
	var allowedBytes []byte
	var deniedBytes []byte

	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &allowedBytes, &deniedBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("skill not found by name: %s", name)
		}
		return nil, fmt.Errorf("get skill by name: %w", err)
	}

	if len(allowedBytes) > 0 {
		if err := json.Unmarshal(allowedBytes, &s.AllowedTools); err != nil {
			return nil, fmt.Errorf("unmarshal allowed tools: %w", err)
		}
	}
	if len(deniedBytes) > 0 {
		if err := json.Unmarshal(deniedBytes, &s.DeniedTools); err != nil {
			return nil, fmt.Errorf("unmarshal denied tools: %w", err)
		}
	}

	servers, err := r.loadMCPServers(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("load mcp servers for skill: %w", err)
	}
	s.MCPServers = servers

	return &s, nil
}

// List retrieves all Skill Collections.
func (r *SkillRepository) List(ctx context.Context) ([]*domain.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, allowed_tools, denied_tools, status, created_at, updated_at
	FROM skills WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	var skills []*domain.Skill
	for rows.Next() {
		var s domain.Skill
		var allowedBytes []byte
		var deniedBytes []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &allowedBytes, &deniedBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}

		if len(allowedBytes) > 0 {
			if err := json.Unmarshal(allowedBytes, &s.AllowedTools); err != nil {
				return nil, fmt.Errorf("unmarshal allowed tools: %w", err)
			}
		}
		if len(deniedBytes) > 0 {
			if err := json.Unmarshal(deniedBytes, &s.DeniedTools); err != nil {
				return nil, fmt.Errorf("unmarshal denied tools: %w", err)
			}
		}

		servers, err := r.loadMCPServers(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("load mcp servers for skill: %w", err)
		}
		s.MCPServers = servers

		skills = append(skills, &s)
	}
	return skills, nil
}

// Update updates an existing Skill Collection properties and its relational MCP server associations.
func (r *SkillRepository) Update(ctx context.Context, s *domain.Skill) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	s.TenantID = ten.FullName()

	allowedJSON, err := json.Marshal(s.AllowedTools)
	if err != nil {
		return fmt.Errorf("marshal allowed tools: %w", err)
	}
	deniedJSON, err := json.Marshal(s.DeniedTools)
	if err != nil {
		return fmt.Errorf("marshal denied tools: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `UPDATE skills SET 
		name = $1, description = $2, allowed_tools = $3, denied_tools = $4, status = $5, updated_at = $6
	WHERE id = $7 AND tenant_id = $8`

	s.UpdatedAt = time.Now().UTC()
	_, err = tx.Exec(ctx, query,
		s.Name, s.Description, allowedJSON, deniedJSON, s.Status, s.UpdatedAt, s.ID, s.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}

	if err := r.saveMCPServers(ctx, tx, s.ID, s.MCPServers); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Delete removes a Skill Collection from the database.
func (r *SkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	// The cascade constraints automatically handle the skill_mcp_servers and agent_skills deletions.
	query := `DELETE FROM skills WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

// AttachSkillToAgent associates a skill collection with an agent UUID.
func (r *SkillRepository) AttachSkillToAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `INSERT INTO agent_skills (agent_id, skill_id)
		SELECT $1, $2 WHERE EXISTS (SELECT 1 FROM skills WHERE id = $2 AND tenant_id = $3)
		ON CONFLICT (agent_id, skill_id) DO NOTHING`
	_, err := r.pool.Exec(ctx, query, agentID, skillID, ten.FullName())
	if err != nil {
		return fmt.Errorf("attach skill to agent: %w", err)
	}
	return nil
}

// DetachSkillFromAgent removes the association between a skill collection and an agent UUID.
func (r *SkillRepository) DetachSkillFromAgent(ctx context.Context, agentID uuid.UUID, skillID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM agent_skills WHERE agent_id = $1 AND skill_id = $2 AND EXISTS (SELECT 1 FROM skills WHERE id = $2 AND tenant_id = $3)`
	_, err := r.pool.Exec(ctx, query, agentID, skillID, ten.FullName())
	if err != nil {
		return fmt.Errorf("detach skill from agent: %w", err)
	}
	return nil
}

// ListSkillsByAgent retrieves all skill collections assigned to a specific agent UUID.
func (r *SkillRepository) ListSkillsByAgent(ctx context.Context, agentID uuid.UUID) ([]*domain.Skill, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		s.id, s.tenant_id, s.name, s.description, s.allowed_tools, s.denied_tools, s.status, s.created_at, s.updated_at
	FROM skills s
	JOIN agent_skills as ON s.id = as.skill_id
	WHERE as.agent_id = $1 AND s.tenant_id = $2
	ORDER BY s.name ASC`

	rows, err := r.pool.Query(ctx, query, agentID, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list skills by agent: %w", err)
	}
	defer rows.Close()

	var skills []*domain.Skill
	for rows.Next() {
		var s domain.Skill
		var allowedBytes []byte
		var deniedBytes []byte

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Description, &allowedBytes, &deniedBytes, &s.Status, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skill by agent: %w", err)
		}

		if len(allowedBytes) > 0 {
			if err := json.Unmarshal(allowedBytes, &s.AllowedTools); err != nil {
				return nil, fmt.Errorf("unmarshal allowed tools: %w", err)
			}
		}
		if len(deniedBytes) > 0 {
			if err := json.Unmarshal(deniedBytes, &s.DeniedTools); err != nil {
				return nil, fmt.Errorf("unmarshal denied tools: %w", err)
			}
		}

		servers, err := r.loadMCPServers(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("load mcp servers for skill: %w", err)
		}
		s.MCPServers = servers

		skills = append(skills, &s)
	}
	return skills, nil
}

// --- Relational helper methods ---

func (r *SkillRepository) loadMCPServers(ctx context.Context, skillID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT mcp_server_id FROM skill_mcp_servers WHERE skill_id = $1`
	rows, err := r.pool.Query(ctx, query, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		servers = append(servers, id)
	}
	return servers, nil
}

func (r *SkillRepository) saveMCPServers(ctx context.Context, tx pgx.Tx, skillID uuid.UUID, serverIDs []uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM skill_mcp_servers WHERE skill_id = $1`, skillID)
	if err != nil {
		return fmt.Errorf("delete old skill mcp servers: %w", err)
	}

	for _, mcpID := range serverIDs {
		_, err := tx.Exec(ctx, `INSERT INTO skill_mcp_servers (skill_id, mcp_server_id) VALUES ($1, $2)`, skillID, mcpID)
		if err != nil {
			return fmt.Errorf("insert skill mcp server: %w", err)
		}
	}
	return nil
}
