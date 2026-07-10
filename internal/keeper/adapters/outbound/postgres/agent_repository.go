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
	"github.com/morphy76/tacito-square/pkg/agentcard"
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
func (r *AgentRepository) Create(ctx context.Context, a *model.Agent) error {
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

	// Check if prompt_template is nil uuid
	var promptTemplate interface{}
	if a.PromptTemplate != uuid.Nil {
		promptTemplate = a.PromptTemplate
	}

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		query := `INSERT INTO agents (
			id, tenant_id, name, description, role, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, community_id, tier, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

		_, err = tx.Exec(ctx, query,
			a.ID, a.TenantID, a.Name, a.Description, "", brainJSON, shortTermJSON, longTermJSON, promptTemplate, mcpClientsJSON, a.Status, a.CommunityID, a.Tier, a.CreatedAt, a.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert agent: %w", err)
		}

		if err := r.saveSkills(ctx, tx, a.ID, a.Skills); err != nil {
			return err
		}
		return nil
	})
}

// GetByID retrieves an Agent template by its ID, loading its associated Skill collection UUIDs.
func (r *AgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, role, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, community_id, tier, created_at, updated_at
	FROM agents WHERE id = $1 AND tenant_id = $2`

	var a model.Agent
	var brainBytes []byte
	var shortBytes []byte
	var longBytes []byte
	var mcpBytes []byte
	var promptTemplate *uuid.UUID

	var deprecatedRole string
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, id, ten.FullName()).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &deprecatedRole, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CommunityID, &a.Tier, &a.CreatedAt, &a.UpdatedAt,
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
func (r *AgentRepository) GetByName(ctx context.Context, name string) (*model.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, role, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, community_id, tier, created_at, updated_at
	FROM agents WHERE name = $1 AND tenant_id = $2`

	var a model.Agent
	var brainBytes []byte
	var shortBytes []byte
	var longBytes []byte
	var mcpBytes []byte
	var promptTemplate *uuid.UUID

	var deprecatedRole string
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, name, ten.FullName()).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &deprecatedRole, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CommunityID, &a.Tier, &a.CreatedAt, &a.UpdatedAt,
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
func (r *AgentRepository) List(ctx context.Context) ([]*model.Agent, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT 
		id, tenant_id, name, description, role, brain, short_term_memory, long_term_memory, prompt_template, mcp_clients, status, community_id, tier, created_at, updated_at
	FROM agents WHERE tenant_id = $1 ORDER BY name ASC`

	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*model.Agent
	for rows.Next() {
		var a model.Agent
		var brainBytes []byte
		var shortBytes []byte
		var longBytes []byte
		var mcpBytes []byte
		var promptTemplate *uuid.UUID

		var deprecatedRole string
		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Description, &deprecatedRole, &brainBytes, &shortBytes, &longBytes, &promptTemplate, &mcpBytes, &a.Status, &a.CommunityID, &a.Tier, &a.CreatedAt, &a.UpdatedAt,
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
func (r *AgentRepository) Update(ctx context.Context, a *model.Agent) error {
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

	var promptTemplate interface{}
	if a.PromptTemplate != uuid.Nil {
		promptTemplate = a.PromptTemplate
	}

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		query := `UPDATE agents SET 
			name = $1, description = $2, brain = $3, short_term_memory = $4, long_term_memory = $5, prompt_template = $6, mcp_clients = $7, status = $8, community_id = $9, tier = $10, updated_at = $11
		WHERE id = $12 AND tenant_id = $13`

		a.UpdatedAt = time.Now().UTC()
		cmdTag, err := tx.Exec(ctx, query,
			a.Name, a.Description, brainJSON, shortTermJSON, longTermJSON, promptTemplate, mcpClientsJSON, a.Status, a.CommunityID, a.Tier, a.UpdatedAt, a.ID, a.TenantID,
		)
		if err != nil {
			return fmt.Errorf("update agent: %w", err)
		}
		if cmdTag.RowsAffected() == 0 {
			return fmt.Errorf("agent not found: %s", a.ID)
		}

		if err := r.saveSkills(ctx, tx, a.ID, a.Skills); err != nil {
			return err
		}
		return nil
	})
}

// Delete removes an Agent template from persistent storage.
func (r *AgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	// agent_skills fk CASCADE constraint handles the associated skill mapping deletion.
	query := `DELETE FROM agents WHERE id = $1 AND tenant_id = $2`
	exec := GetExecutor(ctx, r.pool)
	cmdTag, err := exec.Exec(ctx, query, id, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found: %s", id)
	}
	return nil
}

// --- Relational helper methods ---

func (r *AgentRepository) loadSkills(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT skill_id FROM agent_skills WHERE agent_id = $1`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, agentID)
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

func (r *AgentRepository) saveSkills(ctx context.Context, exec PgxExecutor, agentID uuid.UUID, skillIDs []uuid.UUID) error {
	_, err := exec.Exec(ctx, `DELETE FROM agent_skills WHERE agent_id = $1`, agentID)
	if err != nil {
		return fmt.Errorf("delete old agent skills: %w", err)
	}

	for _, skillID := range skillIDs {
		_, err = exec.Exec(ctx, `INSERT INTO agent_skills (agent_id, skill_id) VALUES ($1, $2)`, agentID, skillID)
		if err != nil {
			return fmt.Errorf("insert agent skill: %w", err)
		}
	}
	return nil
}

// AssignToCommunity binds an Agent template to a Community within a transaction-safe context.
func (r *AgentRepository) AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// 1. Verify community exists and belongs to the tenant, and is active/created
		var commStatus string
		var topology string
		err := tx.QueryRow(ctx, `SELECT status, topology FROM communities WHERE id = $1 AND tenant_id = $2`, communityID, ten.FullName()).Scan(&commStatus, &topology)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("community not found: %s", communityID)
			}
			return fmt.Errorf("check community: %w", err)
		}
		if commStatus != "active" && commStatus != "created" {
			return fmt.Errorf("community status is %s, must be active or created", commStatus)
		}

		// 2. Verify agent exists, belongs to tenant, and is not already assigned
		var currentCommID *uuid.UUID
		var agentStatus string
		var agentRole string
		err = tx.QueryRow(ctx, `SELECT community_id, status, role FROM agents WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, agentID, ten.FullName()).Scan(&currentCommID, &agentStatus, &agentRole)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("agent not found: %s", agentID)
			}
			return fmt.Errorf("check agent: %w", err)
		}

		if currentCommID != nil {
			return fmt.Errorf("agent already assigned to community: %s", currentCommID)
		}

		// Verify topology constraints
		if topology == string(model.CommunityTopologySingleAgent) {
			var count int
			err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE community_id = $1`, communityID).Scan(&count)
			if err != nil {
				return fmt.Errorf("check community agents count: %w", err)
			}
			if count >= 1 {
				return fmt.Errorf("community with single-agent topology cannot have more than one agent assigned")
			}
		} else if topology == string(model.CommunityTopologyHubSpoke) {
			if agentRole == "hub" {
				var hubCount int
				err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE community_id = $1 AND role = 'hub'`, communityID).Scan(&hubCount)
				if err != nil {
					return fmt.Errorf("check community hubs count: %w", err)
				}
				if hubCount >= 1 {
					return fmt.Errorf("community with hub-spoke topology cannot have more than one hub agent assigned")
				}
			}
		}

		// 3. Update agent assignment
		updatedAt := time.Now().UTC()
		_, err = tx.Exec(ctx, `UPDATE agents SET community_id = $1, status = $2, updated_at = $3 WHERE id = $4 AND tenant_id = $5`,
			communityID, string(model.AgentStatusPending), updatedAt, agentID, ten.FullName())
		if err != nil {
			return fmt.Errorf("update agent assignment: %w", err)
		}

		return nil
	})
}

// UnassignFromCommunity removes an Agent template assignment from a Community.
func (r *AgentRepository) UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	return ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		// Verify agent exists, belongs to tenant, and community_id matches
		var currentCommID *uuid.UUID
		var agentStatus string
		err := tx.QueryRow(ctx, `SELECT community_id, status FROM agents WHERE id = $1 AND tenant_id = $2 FOR UPDATE`, agentID, ten.FullName()).Scan(&currentCommID, &agentStatus)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("agent not found: %s", agentID)
			}
			return fmt.Errorf("check agent: %w", err)
		}

		if currentCommID == nil || *currentCommID != communityID {
			return fmt.Errorf("agent is not assigned to community: %s", communityID)
		}

		// Update agent to clear assignment
		updatedAt := time.Now().UTC()
		_, err = tx.Exec(ctx, `UPDATE agents SET community_id = NULL, status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
			string(model.AgentStatusDefined), updatedAt, agentID, ten.FullName())
		if err != nil {
			return fmt.Errorf("clear agent assignment: %w", err)
		}

		return nil
	})
}

// UpdateStatus updates an agent's status in the database.
func (r *AgentRepository) UpdateStatus(ctx context.Context, agentID uuid.UUID, status model.AgentStatus) (bool, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return false, errors.New("tenant resolution failed")
	}

	query := `UPDATE agents SET status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND status != $1`

	exec := GetExecutor(ctx, r.pool)
	cmdTag, err := exec.Exec(ctx, query, string(status), time.Now().UTC(), agentID, ten.FullName())
	if err != nil {
		return false, fmt.Errorf("update agent status: %w", err)
	}
	return cmdTag.RowsAffected() > 0, nil
}

// UpsertRegistration registers an agent's active card.
func (r *AgentRepository) UpsertRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID, card *agentcard.AgentCard) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal agent card: %w", err)
	}

	query := `INSERT INTO agent_registrations (
		agent_id, community_id, tenant_id, card, last_seen_at
	) VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (agent_id, community_id) DO UPDATE SET
		card = EXCLUDED.card,
		last_seen_at = EXCLUDED.last_seen_at`

	exec := GetExecutor(ctx, r.pool)
	now := time.Now().UTC()
	_, err = exec.Exec(ctx, query, agentID, communityID, ten.FullName(), cardJSON, now)
	if err != nil {
		return fmt.Errorf("upsert agent registration: %w", err)
	}
	return nil
}

// GetRegistration retrieves a registered agent's card.
func (r *AgentRepository) GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*agentcard.AgentCard, time.Time, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, time.Time{}, errors.New("tenant resolution failed")
	}

	query := `SELECT card, last_seen_at FROM agent_registrations 
	WHERE agent_id = $1 AND community_id = $2 AND tenant_id = $3`

	var cardBytes []byte
	var lastSeen time.Time
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, agentID, communityID, ten.FullName()).Scan(&cardBytes, &lastSeen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, time.Time{}, fmt.Errorf("registration not found for agent %s in community %s", agentID, communityID)
		}
		return nil, time.Time{}, fmt.Errorf("get agent registration: %w", err)
	}

	var card agentcard.AgentCard
	if err := json.Unmarshal(cardBytes, &card); err != nil {
		return nil, time.Time{}, fmt.Errorf("unmarshal agent card: %w", err)
	}
	card.AgentID = agentID.String()
	return &card, lastSeen, nil
}

// GetActiveRegistrationsByCommunity retrieves registrations inside a community.
func (r *AgentRepository) GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*agentcard.AgentCard, time.Time, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, time.Time{}, errors.New("tenant resolution failed")
	}

	query := `SELECT agent_id, card, last_seen_at FROM agent_registrations 
	WHERE community_id = $1 AND tenant_id = $2 ORDER BY last_seen_at DESC`

	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, communityID, ten.FullName())
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("get active registrations by community: %w", err)
	}
	defer rows.Close()

	var cards []*agentcard.AgentCard
	var latestTime time.Time
	for rows.Next() {
		var agentID uuid.UUID
		var cardBytes []byte
		var lastSeen time.Time
		if err := rows.Scan(&agentID, &cardBytes, &lastSeen); err != nil {
			return nil, time.Time{}, fmt.Errorf("scan agent card: %w", err)
		}
		if lastSeen.After(latestTime) {
			latestTime = lastSeen
		}
		var card agentcard.AgentCard
		if err := json.Unmarshal(cardBytes, &card); err != nil {
			return nil, time.Time{}, fmt.Errorf("unmarshal agent card list item: %w", err)
		}
		card.AgentID = agentID.String()
		cards = append(cards, &card)
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, fmt.Errorf("rows iteration error: %w", err)
	}
	return cards, latestTime, nil
}

// PruneStaleRegistrations deletes registrations missing heartbeats and sets agent status to offline.
func (r *AgentRepository) PruneStaleRegistrations(ctx context.Context, threshold time.Duration) ([]agentcard.AgentCommunityRef, error) {
	ten := tenant.FromContext(ctx)

	cutoff := time.Now().UTC().Add(-threshold)

	type prunedRow struct {
		agentID     uuid.UUID
		communityID uuid.UUID
		tenantID    string
	}
	var prunedRows []prunedRow

	err := ExecuteInTxOrPool(ctx, r.pool, func(tx pgx.Tx) error {
		var rows pgx.Rows
		var err error

		if ten != nil {
			queryDel := `DELETE FROM agent_registrations 
			WHERE last_seen_at < $1 AND tenant_id = $2
			RETURNING agent_id, community_id, tenant_id`
			rows, err = tx.Query(ctx, queryDel, cutoff, ten.FullName())
		} else {
			queryDel := `DELETE FROM agent_registrations 
			WHERE last_seen_at < $1
			RETURNING agent_id, community_id, tenant_id`
			rows, err = tx.Query(ctx, queryDel, cutoff)
		}

		if err != nil {
			return fmt.Errorf("delete stale registrations: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var rRow prunedRow
			if err := rows.Scan(&rRow.agentID, &rRow.communityID, &rRow.tenantID); err != nil {
				return fmt.Errorf("scan pruned registration: %w", err)
			}
			prunedRows = append(prunedRows, rRow)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("rows error: %w", err)
		}

		// 2. Set status to offline (stopped) in agents table
		if len(prunedRows) > 0 {
			// Group by tenant for segregated updates
			prunedByTenant := make(map[string][]uuid.UUID)
			for _, row := range prunedRows {
				prunedByTenant[row.tenantID] = append(prunedByTenant[row.tenantID], row.agentID)
			}

			for tID, agentIDs := range prunedByTenant {
				queryUpdate := `UPDATE agents SET status = $1, updated_at = $2 
				WHERE id = ANY($3) AND tenant_id = $4`
				
				_, err = tx.Exec(ctx, queryUpdate, string(model.AgentStatusStopped), time.Now().UTC(), agentIDs, tID)
				if err != nil {
					return fmt.Errorf("update pruned agents status: %w", err)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	prunedRefs := make([]agentcard.AgentCommunityRef, 0, len(prunedRows))
	for _, row := range prunedRows {
		prunedRefs = append(prunedRefs, agentcard.AgentCommunityRef{
			AgentID:     row.agentID.String(),
			CommunityID: row.communityID.String(),
			TenantID:    row.tenantID,
		})
	}
	return prunedRefs, nil
}

// DeleteRegistration deletes a registered agent card for a community.
func (r *AgentRepository) DeleteRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM agent_registrations 
	WHERE agent_id = $1 AND community_id = $2 AND tenant_id = $3`

	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, agentID, communityID, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete agent registration: %w", err)
	}
	return nil
}

