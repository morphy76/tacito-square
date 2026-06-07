# TASK-M6.5-T1: Database Schema & Models

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.5-T1                          |
| Spec        | SPEC-FR-M6.5                          |
| Boundary    | Database & Models                     |
| Status      | DRAFT                                 |
| Depends On  | —                                     |

## Objective

Modify the database migration to add the `card` column to the `agents` table, implement the domain models for `AgentCard` and `CommunityCard` in the respective component domain layers, and update the repository layer to query and persist card data.

## Files

| File | Action |
|------|--------|
| `deploy/postgres/migrations/00001_init.sql` | MODIFY |
| `internal/keeper/domain/model/agent.go` | MODIFY |
| `internal/keeper/domain/model/agent_card.go` | NEW |
| `internal/keeper/domain/model/community_card.go` | NEW |
| `internal/agent/domain/model/agent_card.go` | NEW |
| `internal/keeper/adapters/outbound/postgres/agent_repository.go` | MODIFY |
| `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` | MODIFY |

## RED Phase

1. Add a test case `TestAgentRepository_CreateAndGetWithCard` in `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` that instantiates an agent with a non-empty `Card` object, saves it, retrieves it, and asserts the card values match.
2. Verify that `make test` fails to compile because the `Card` property is missing from the `model.Agent` struct, and the DB schema does not support it.

## GREEN Phase

1. Modify [00001_init.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00001_init.sql) to add a `card JSONB` column to the `agents` table definition.
2. Define the `AgentCard`, `AgentCardCapabilities`, `AgentCardAuthentication`, and `AgentCardSkill` structures in `internal/keeper/domain/model/agent_card.go` and `internal/agent/domain/model/agent_card.go`.
3. Define the `CommunityCard` struct in `internal/keeper/domain/model/community_card.go`.
4. Update `internal/keeper/domain/model/agent.go` to include the `Card *AgentCard` field and update its `Validate()` method (validation logic should ensure card is valid if present).
5. Update `internal/keeper/adapters/outbound/postgres/agent_repository.go` to support inserting, updating, and selecting the `card` JSONB column (serialize to JSON string on write, deserialize on read).
6. Run `make test` and verify tests pass.

## REFACTOR Phase

- Confirm JSON serialization does not leak raw database errors.
- Ensure SQL query statements are clean and trace spans are logged correctly.
