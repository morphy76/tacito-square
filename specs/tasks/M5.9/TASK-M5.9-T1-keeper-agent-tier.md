# TASK-M5.9-T1: Keeper Agent Domain, REST, and DB Schema Update for Tiers

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.9-T1                                |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.9                                |
| Depends On    | none                                        |

## Description

Upgrades the Keeper component's Agent model, database migrations, and REST handlers to accept and store the optional `tier` field.

## Work Items

1. **RED Phase**:
   - Write unit tests in [agent_handlers_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers_test.go) asserting that `POST /api/v1/agents` and `PUT /api/v1/agents/:id` request bodies successfully parse and map the optional `tier` string parameter.
   - Verify that the tests fail when run against the existing codebase.
2. **GREEN Phase**:
   - Add `Tier` string field (`json:"tier"`) to `Agent` domain model in [agent.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/domain/model/agent.go).
   - Create a new goose SQL migration file inside the Keeper migrations directory to add `tier` column (`varchar(50)`) to `agents` table.
   - Update Keeper REST request structs in [agent_handlers.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/inbound/http/agent_handlers.go) to bind the optional `tier` parameter.
   - Update the Agent schema in [openapi.json](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/openapi.json) to document the new `tier` property.
   - Verify all tests compile and pass successfully.
3. **REFACTOR Phase**:
   - Review Keeper GORM/postgres repository mappings to ensure `tier` is cleanly updated and loaded without errors.

## Acceptance Criteria

1. Creating or updating an agent via REST successfully saves and retrieves the `tier` field.
2. The OpenAPI spec is updated with the new property and successfully passes the contract parity test suite.
