# TASK-M3.5.1: Agent Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.5.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.5                                |
| Depends On    | none                                        |

## Description

Define the `Agent` aggregate domain model, sub-configurations validation business rules, repository interfaces, and GORM database persistence adapter for Agent templates. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create domain unit tests in `internal/keeper/domain/agent_test.go` verifying:
     - Basic validations (mandatory fields: `name`, `status`, `tenant_id`).
     - Struct invariants for the nested configurations:
       - `Brain` (model, temperature, max tokens, endpoint, credentials).
       - `ShortTermMemory` (key namespaces, TTL).
       - `LongTermMemory` (collection name, vector dimension).
       - `MCPClient` (server connection parameters/config).
     - Lifecycle status transition constraints (e.g., valid statuses: `defined`, `assigned`, `active`, `terminated`).
   - Create database integration tests in `internal/keeper/adapters/postgres/agent_repository_test.go` verifying:
     - CRUD operations on persistent storage.
     - Scope filtering by `tenant_id` fetched from context (ensuring strict multi-tenant isolation).
     - Association mapping and table joins (e.g., retrieving skills, prompt template references, MCP client configs).
2. **GREEN Phase**:
   - Create `internal/keeper/domain/agent.go` defining the `Agent` aggregate structure, configuration schemas, and validation logic.
   - Register the `AgentRepository` port in `internal/keeper/application/ports/outbound/repositories.go`.
   - Create database schema migrations in `deploy/postgres/migrations/` defining tables for agent configurations, ensuring non-nullable `tenant_id VARCHAR(255) NOT NULL` and unique constraints qualified by `(tenant_id, name)`.
   - Implement repository persistence adapter in `internal/keeper/adapters/postgres/agent_repository.go`.
3. **REFACTOR Phase**:
   - Refactor nested configuration handling (e.g., serialize configs using JSON format or normalize database schema correctly).
   - Ensure the hexagonal core (`internal/keeper/domain`) is completely isolated and decoupled from postgres or gin packages.

## Acceptance Criteria

1. `agent_test.go` and `agent_repository_test.go` pass successfully.
2. The domain model, schema migrations, and repository persistence adapter are fully implemented and free of hexagonal leaks.
