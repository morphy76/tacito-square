# TASK-M6.5-T1: Database Schema & Models

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.5-T1                          |
| Spec        | SPEC-FR-M6.5                          |
| Boundary    | Database & Models                     |
| Status      | VERIFIED                              |
| Depends On  | —                                     |

## Objective

Modify the database migration to create a new `agent_registrations` table mapping agent ID and community ID to its card details and heartbeat timestamp, define the domain models for `AgentCard` and `CommunityCard` in the respective domain layers, and implement the repository queries to save, fetch, and delete registrations.

## Files

| File | Action |
|------|--------|
| `deploy/postgres/migrations/00001_init.sql` | MODIFY |
| `internal/keeper/domain/model/agent_card.go` | NEW |
| `internal/keeper/domain/model/community_card.go` | NEW |
| `internal/agent/domain/model/agent_card.go` | NEW |
| `internal/keeper/adapters/outbound/postgres/agent_repository.go` | MODIFY |
| `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` | MODIFY |

## RED Phase

1. Add a test case `TestAgentRepository_SaveAndGetRegistration` in `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` that saves a new `AgentCard` associated with a valid agent and community in the `agent_registrations` table, retrieves it, and asserts card details and `last_seen_at` matches.
2. Verify that `make test` fails because the `agent_registrations` table and corresponding repository methods are missing (RED).

## GREEN Phase

1. Modify [00001_init.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00001_init.sql) to add the `agent_registrations` table with a composite primary key:
   ```sql
   CREATE TABLE IF NOT EXISTS agent_registrations (
       agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
       community_id UUID NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
       tenant_id VARCHAR(255) NOT NULL,
       card JSONB NOT NULL,
       last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL,
       PRIMARY KEY (agent_id, community_id)
   );
   CREATE INDEX IF NOT EXISTS idx_agent_registrations_tenant_id ON agent_registrations(tenant_id);
   CREATE INDEX IF NOT EXISTS idx_agent_registrations_community ON agent_registrations(community_id);
   ```
2. Define the `AgentCard` and nested structs in `internal/keeper/domain/model/agent_card.go` and `internal/agent/domain/model/agent_card.go`.
3. Define the `CommunityCard` struct in `internal/keeper/domain/model/community_card.go`.
4. Implement repository methods in `internal/keeper/adapters/outbound/postgres/agent_repository.go` to handle:
   - `UpsertRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID, card *model.AgentCard) error`
   - `GetRegistration(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) (*model.AgentCard, error)`
   - `GetActiveRegistrationsByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.AgentCard, error)`
   - `PruneStaleRegistrations(ctx context.Context, threshold time.Duration) ([]model.AgentCommunityRef, error)` (where `AgentCommunityRef` maps agent ID and community ID of pruned entries, to allow publishing precise NATS status events).
5. Verify that tests compile and pass (GREEN).

## REFACTOR Phase

- Check that transaction handles cascading deletes or status updates properly.
- Ensure that the query scanner and unmarshaling are clean and trace spans are logged correctly.
