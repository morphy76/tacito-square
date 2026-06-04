# TASK-M5.9-T1: Keeper Agent Domain, REST, and DB Schema Update for Tiers

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T1                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | none                                        |

## Description

Upgrades the Keeper component's Agent model, database migrations, pgx repository queries, and REST handlers to accept and store the optional `tier` field.

## Work Items

1. **RED Phase**:
   - Write unit tests in [agent_handlers_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers_test.go) asserting that `POST /api/v1/agents` and `PUT /api/v1/agents/:id` request bodies successfully parse and map the optional `deployment.tier` string nested under a `deployment` sub-object.
   - Verify that the tests fail when run against the existing codebase.
2. **GREEN Phase**:
   - Add `Tier string` field (`json:"tier"`) to the `Agent` domain model in [agent.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/domain/model/agent.go).
   - Create a new goose SQL migration [00002_add_agent_tier.sql](file:///Users/R.Pasquini/Projects/side/tacito-square/deploy/postgres/migrations/00002_add_agent_tier.sql) adding `tier VARCHAR(50) NOT NULL DEFAULT ''` to the `agents` table.
   - Add a `DeploymentRequest` sub-struct with a `Tier` field to `CreateAgentRequest` and `UpdateAgentRequest` in [agent_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers.go); map it through to the `Agent` domain model.
   - Update all SELECT, INSERT, and UPDATE queries in [agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) to include the `tier` column.
   - Update the Agent create/update schema in [openapi.json](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/openapi.json) to document the optional `deployment.tier` property.
   - Verify all tests compile and pass successfully.
3. **REFACTOR Phase**:
   - Review pgx Scan/Exec parameter lists for consistency across all query methods.

## Acceptance Criteria

1. Creating or updating an agent via REST with `{ "deployment": { "tier": "heavy" } }` successfully saves and retrieves the `tier` field from PostgreSQL.
2. Creating or updating an agent without a `deployment` block saves `tier` as empty string.
3. The OpenAPI spec documents the new `deployment.tier` property and passes the contract parity test suite.
