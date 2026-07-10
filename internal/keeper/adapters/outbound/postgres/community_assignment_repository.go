package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// CommunityAssignmentRepository implements the outbound.CommunityAssignmentRepository interface.
type CommunityAssignmentRepository struct {
	pool *pgxpool.Pool
}

// NewCommunityAssignmentRepository creates a new instance of CommunityAssignmentRepository.
func NewCommunityAssignmentRepository(pool *pgxpool.Pool) *CommunityAssignmentRepository {
	return &CommunityAssignmentRepository{pool: pool}
}

// Create inserts a new CommunityAssignment into the database.
func (r *CommunityAssignmentRepository) Create(ctx context.Context, ca *model.CommunityAssignment) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}
	ca.TenantID = ten.FullName()

	query := `INSERT INTO community_assignments (
		community_id, agent_id, tenant_id, role, informed_at, assigned_at
	) VALUES ($1, $2, $3, $4, $5, $6)`

	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query,
		ca.CommunityID, ca.AgentID, ca.TenantID, string(ca.Role), ca.InformedAt, ca.AssignedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return fmt.Errorf("agent %s is already assigned to community %s: %w", ca.AgentID, ca.CommunityID, err)
		}
		return fmt.Errorf("create community assignment: %w", err)
	}
	return nil
}

// Delete removes an assignment record from the database.
func (r *CommunityAssignmentRepository) Delete(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID) error {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return errors.New("tenant resolution failed")
	}

	query := `DELETE FROM community_assignments 
	WHERE community_id = $1 AND agent_id = $2 AND tenant_id = $3`

	exec := GetExecutor(ctx, r.pool)
	cmdTag, err := exec.Exec(ctx, query, communityID, agentID, ten.FullName())
	if err != nil {
		return fmt.Errorf("delete community assignment: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found: community %s, agent %s", communityID, agentID)
	}
	return nil
}

// ListByCommunity retrieves all assignments for a given community.
func (r *CommunityAssignmentRepository) ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return nil, errors.New("tenant resolution failed")
	}

	query := `SELECT community_id, agent_id, tenant_id, role, informed_at, assigned_at 
	FROM community_assignments 
	WHERE community_id = $1 AND tenant_id = $2 ORDER BY assigned_at ASC`

	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, communityID, ten.FullName())
	if err != nil {
		return nil, fmt.Errorf("list community assignments: %w", err)
	}
	defer rows.Close()

	var list []*model.CommunityAssignment
	for rows.Next() {
		ca, err := r.scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, ca)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return list, nil
}

// CountHubs returns the number of hubs assigned to a community.
func (r *CommunityAssignmentRepository) CountHubs(ctx context.Context, communityID uuid.UUID) (int, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return 0, errors.New("tenant resolution failed")
	}

	query := `SELECT COUNT(*) FROM community_assignments 
	WHERE community_id = $1 AND tenant_id = $2 AND role = $3`

	var count int
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, communityID, ten.FullName(), string(model.AgentRoleHub)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count hubs: %w", err)
	}
	return count, nil
}

// CountByCommunity returns the total number of assignments to a community.
func (r *CommunityAssignmentRepository) CountByCommunity(ctx context.Context, communityID uuid.UUID) (int, error) {
	ten := tenant.FromContext(ctx)
	if ten == nil {
		return 0, errors.New("tenant resolution failed")
	}

	query := `SELECT COUNT(*) FROM community_assignments 
	WHERE community_id = $1 AND tenant_id = $2`

	var count int
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, communityID, ten.FullName()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by community: %w", err)
	}
	return count, nil
}

// scanAssignment scans a single row from pgx.Rows into a CommunityAssignment aggregate. (REFACTOR Phase)
func (r *CommunityAssignmentRepository) scanAssignment(rows pgx.Rows) (*model.CommunityAssignment, error) {
	var ca model.CommunityAssignment
	var roleStr string
	err := rows.Scan(
		&ca.CommunityID, &ca.AgentID, &ca.TenantID, &roleStr, &ca.InformedAt, &ca.AssignedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan community assignment: %w", err)
	}
	ca.Role = model.AgentRole(roleStr)
	return &ca, nil
}
