# TASK-M6.5.1.3: Keeper Postgres Adapter & DB Migration — community_assignments

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.1.3 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.1 |
| Depends On  | TASK-M6.5.1.2 |

## Description

Implement the infrastructure layer for the `CommunityAssignment` boundary. This covers two sub-boundaries:
- **DB migration**: add the `community_assignments` table (with `tenant_id`, `informed_at`, and secondary indexes) while retaining the deprecated `agents.role` column with a backfill trigger.
- **Postgres adapter**: implement `CommunityAssignmentRepository` using `pgx`, with tenant-scoped queries on all operations.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/adapters/outbound/postgres/community_assignment_repository_test.go` using testcontainers (PostgreSQL):
     - Test `Create` inserts a row with correct `tenant_id`, `role`, `assigned_at`.
     - Test `Create` returns a conflict/unique-violation error for a duplicate `(community_id, agent_id)`.
     - Test `ListByCommunity` returns all assignments for a community scoped to `tenant_id`.
     - Test `CountHubs` returns the correct count of `role = 'hub'` rows.
     - Test `CountByCommunity` returns the total count for a given `community_id`.
     - Test `Delete` removes the row and returns not-found error when row is absent.

2. **GREEN Phase**:
   - Create a new goose migration file (next sequence number after existing migrations):
     ```sql
     -- +goose Up
     CREATE TABLE community_assignments (
         community_id  UUID          NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
         agent_id      UUID          NOT NULL REFERENCES agents(id)      ON DELETE CASCADE,
         tenant_id     VARCHAR(255)  NOT NULL,
         role          TEXT          NOT NULL,
         informed_at   TIMESTAMPTZ,
         assigned_at   TIMESTAMPTZ   NOT NULL,
         PRIMARY KEY (community_id, agent_id)
     );
     CREATE INDEX idx_community_assignments_agent_id  ON community_assignments(agent_id);
     CREATE INDEX idx_community_assignments_tenant_id ON community_assignments(tenant_id);

     -- Deprecated backward-compat: keep agents.role populated via trigger
     CREATE OR REPLACE FUNCTION sync_agent_role_from_assignment()
     RETURNS TRIGGER LANGUAGE plpgsql AS $$
     BEGIN
         UPDATE agents SET role = NEW.role WHERE id = NEW.agent_id;
         RETURN NEW;
     END;
     $$;
     CREATE TRIGGER trg_sync_agent_role
     AFTER INSERT OR UPDATE ON community_assignments
     FOR EACH ROW EXECUTE FUNCTION sync_agent_role_from_assignment();

     -- +goose Down
     DROP TRIGGER IF EXISTS trg_sync_agent_role ON community_assignments;
     DROP FUNCTION IF EXISTS sync_agent_role_from_assignment();
     DROP TABLE IF EXISTS community_assignments;
     ```
   - Create `internal/keeper/adapters/outbound/postgres/community_assignment_repository.go`:
     - Implement `CommunityAssignmentRepository` interface with pgx queries.
     - All queries filter by `tenant_id` extracted from `context.Context` via the tenant helper.
     - Wrap pgx errors (unique violation → domain conflict error, no rows → domain not-found error).

3. **REFACTOR Phase**:
   - Extract a shared `scanAssignment` helper to avoid row-scanning duplication across `ListByCommunity` and any future GetByID.
   - Ensure all queries use parameterised pgx arguments (no string interpolation).

## Acceptance Criteria

1. Testcontainer integration tests all pass GREEN with a real PostgreSQL instance.
2. All `community_assignments` queries include `tenant_id` in `WHERE` clauses.
3. Migration runs cleanly forward (`goose up`) and backward (`goose down`) without errors.
4. The backfill trigger correctly updates `agents.role` on `INSERT`/`UPDATE` to `community_assignments`.
5. The pgx adapter implements the full `CommunityAssignmentRepository` interface — no methods missing.
