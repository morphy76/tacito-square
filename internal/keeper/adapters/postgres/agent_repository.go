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

// AgentRepository implements the outbound.AgentRepository port interface using PostgreSQL.
type AgentRepository struct {
	pool *pgxpool.Pool
}

// NewAgentRepository creates a new instance of AgentRepository.
func NewAgentRepository(pool *pgxpool.Pool) *AgentRepository {
	return &AgentRepository{pool: pool}
}

// Create inserts a new Agent template and its associated skill references into the database.
func (r *AgentRepository) Create(ctx context.Context, a *domain.Agent) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	a.TenantID = ten.FullName()

	brainJSON, err := json.Marshal(a.Brain)
	if err != nil {
		return fmt.Errorf("marshal brain config: %w", err)
	}

	shortTermJSON, err := json.Marshal(a.ShortTermMemory)
	if err != nil {
		return fmt.Errorf("marshal short-term memory config: %w", err)
	}

	longTermJSON, err := json.Marshal(a.LongTermMemory)
	if err != nil {
		return fmt.Errorf("marshal long-term memory config: %w", err)
	}

	mcpClientsJSON, err := json.Marshal(a.MCPClients)
	if err != nil {
		return fmt.Errorf("marshal mcp clients config: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if prompt_template is nil uuid
	var promptTemplate interface{}
	if a.PromptTemplate != uuid.Nil {
		promptTemplate = a.PromptTemplate
	}

	query := `INSERT INTO agents (
		id, tenant_id, name, description, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = tx.Exec(ctx, query,
		a.ID, a.TenantID, a.Name, a.Description, brainJSON, shortTermJSON, longTermJSON, promptTemplate, mcpClientsJSON, a.Status, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert agent: %w", err)
	}

	if err := r.saveSkills(ctx, tx, a.ID, a.Skills); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// GetByID retrieves an Agent template by its ID, loading its associated Skill collection UUIDs.
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, created_at, updated_at
	FROM agents WHERE id = $1 AND tenant_id = $2`

	var a domain.Agent
	var brainBytes []byte
	var shortBytes []byte
	var longBytes []byte
	var mcpBytes []byte
	var promptTemplate *uuid.UUID

	err := r.pool.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("agent not found: %s", id)
		}
		return nil, fmt.Errorf("get agent by id: %w", err)
	}

	if promptTemplate != nil {
		a.PromptTemplate = *promptTemplate
	}

	if err := json.Unmarshal(brainBytes, &a.Brain); err != nil {
		return nil, fmt.Errorf("unmarshal brain config: %w", err)
	}
	if err := json.Unmarshal(shortBytes, &a.ShortTermMemory); err != nil {
		return nil, fmt.Errorf("unmarshal short-term memory: %w", err)
	}
	if err := json.Unmarshal(longBytes, &a.LongTermMemory); err != nil {
		return nil, fmt.Errorf("unmarshal long-term memory: %w", err)
	}
	if err := json.Unmarshal(mcpBytes, &a.MCPClients); err != nil {
		return nil, fmt.Errorf("unmarshal mcp clients: %w", err)
	}

	skills, err := r.loadSkills(ctx, a.ID)
	if err != nil {
		return nil, fmt.Errorf("load skills for agent: %w", err)
	}
	a.Skills = skills

	return &a, nil
}

// GetByName retrieves an Agent template by its unique name.
func (r *AgentRepository) GetByName(ctx context.Context, name string) (*domain.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, created_at, updated_at
	FROM agents WHERE name = $1 AND tenant_id = $2`

	var a domain.Agent
	var brainBytes []byte
	var shortBytes []byte
	var longBytes []byte
	var mcpBytes []byte
	var promptTemplate *uuid.UUID

	err := r.pool.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("agent not found by name: %s", name)
		}
		return nil, fmt.Errorf("get agent by name: %w", err)
	}

	if promptTemplate != nil {
		a.PromptTemplate = *promptTemplate
	}

	if err := json.Unmarshal(brainBytes, &a.Brain); err != nil {
		return nil, fmt.Errorf("unmarshal brain config: %w", err)
	}
	if err := json.Unmarshal(shortBytes, &a.ShortTermMemory); err != nil {
		return nil, fmt.Errorf("unmarshal short-term memory: %w", err)
	}
	if err := json.Unmarshal(longBytes, &a.LongTermMemory); err != nil {
		return nil, fmt.Errorf("unmarshal long-term memory: %w", err)
	}
	if err := json.Unmarshal(mcpBytes, &a.MCPClients); err != nil {
		return nil, fmt.Errorf("unmarshal mcp clients: %w", err)
	}

	skills, err := r.loadSkills(ctx, a.ID)
	if err != nil {
		return nil, fmt.Errorf("load skills for agent: %w", err)
	}
	a.Skills = skills

	return &a, nil
}

// List retrieves all Agent templates for the dynamic tenant context.
func (r *AgentRepository) List(ctx context.Context) ([]*domain.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, created_at, updated_at
	FROM agents WHERE tenant_id = $1 ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*domain.Agent
	for rows.Next() {
		var a domain.Agent
		var brainBytes []byte
		var shortBytes []byte
		var longBytes []byte
		var mcpBytes []byte
		var promptTemplate *uuid.UUID

		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Description, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}

		if promptTemplate != nil {
			a.PromptTemplate = *promptTemplate
		}

		if err := json.Unmarshal(brainBytes, &a.Brain); err != nil {
			return nil, fmt.Errorf("unmarshal brain config: %w", err)
		}
		if err := json.Unmarshal(shortBytes, &a.ShortTermMemory); err != nil {
			return nil, fmt.Errorf("unmarshal short-term memory: %w", err)
		}
		if err := json.Unmarshal(longBytes, &a.LongTermMemory); err != nil {
			return nil, fmt.Errorf("unmarshal long-term memory: %w", err)
		}
		if err := json.Unmarshal(mcpBytes, &a.MCPClients); err != nil {
			return nil, fmt.Errorf("unmarshal mcp clients: %w", err)
		}

		skills, err := r.loadSkills(ctx, a.ID)
		if err != nil {
			return nil, fmt.Errorf("load skills for agent: %w", err)
		}
		a.Skills = skills

		agents = append(agents, &a)
	}
	return agents, nil
}

// Update updates an existing Agent template's properties and its skill mapping relationships.
func (r *AgentRepository) Update(ctx context.Context, a *domain.Agent) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	a.TenantID = ten.FullName()

	brainJSON, err := json.Marshal(a.Brain)
	if err != nil {
		return fmt.Errorf("marshal brain config: %w", err)
	}

	shortTermJSON, err := json.Marshal(a.ShortTermMemory)
	if err != nil {
		return fmt.Errorf("marshal short-term memory config: %w", err)
	}

	longTermJSON, err := json.Marshal(a.LongTermMemory)
	if err != nil {
		return fmt.Errorf("marshal long-term memory config: %w", err)
	}

	mcpClientsJSON, err := json.Marshal(a.MCPClients)
	if err != nil {
		return fmt.Errorf("marshal mcp clients config: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var promptTemplate interface{}
	if a.PromptTemplate != uuid.Nil {
		promptTemplate = a.PromptTemplate
	}

	query := `UPDATE agents SET 
		name = $1, description = $2, brain = $3, short_term_memory = $4, long_term_memory = $5, prompt_template = $6, mcp_clients = $7, status = $8, updated_at = $9
	WHERE id = $10 AND tenant_id = $11`

	a.UpdatedAt = time.Now().UTC()
	_, err = tx.Exec(ctx, query,
		a.Name, a.Description, brainJSON, shortTermJSON, longTermJSON, promptTemplate, mcpClientsJSON, a.Status, a.UpdatedAt, a.ID, a.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update agent: %w", err)
	}

	if err := r.saveSkills(ctx, tx, a.ID, a.Skills); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Delete removes an Agent template from persistent storage.
func (r *AgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	// agent_skills fk CASCADE constraint handles the associated skill mapping deletion.
	query := `DELETE FROM agents WHERE id = $1 AND tenant_id = $2`
	_, err := r.pool.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	return nil
}

// --- Relational helper methods ---

func (r *AgentRepository) loadSkills(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT skill_id FROM agent_skills WHERE agent_id = $1`
	rows, err := r.pool.Query(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		skills = append(skills, id)
	}
	return skills, nil
}

func (r *AgentRepository) saveSkills(ctx context.Context, tx pgx.Tx, agentID uuid.UUID, skillIDs []uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM agent_skills WHERE agent_id = $1`, agentID)
	if err != nil {
		return fmt.Errorf("delete old agent skills: %w", err)
	}

	for _, skillID := range skillIDs {
		_, err := tx.Exec(ctx, `INSERT INTO agent_skills (agent_id, skill_id) VALUES ($1, $2)`, agentID, skillID)
		if err != nil {
			return fmt.Errorf("insert agent skill: %w", err)
		}
	}
	return nil
}
