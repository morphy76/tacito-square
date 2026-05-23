# TASK-M3.2.1: MCP Servers Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.2.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M3.2                                |
| Depends On    | none                                        |

## Description

Define the domain model, transport-specific validation business rules, repository interfaces, and GORM database persistence adapter for MCP Servers. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/domain/mcp_server_test.go` verifying:
     - Basic validations (mandatory name, description length).
     - Transport constraints: `stdio` transport requires a valid `command` executable path; `sse` transport requires a valid, parsed `url`.
   - Create database integration tests in `internal/keeper/adapters/postgres/mcp_server_repository_test.go` verifying CRUD operations on persistent storage.
2. **GREEN Phase**:
   - Create `internal/keeper/domain/mcp_server.go` containing the `MCPServer` aggregate struct and custom validation/transport verification logic.
   - Define database schema migration in `deploy/postgres/migrations/` (defining table for MCP Servers).
   - Register repository port in `internal/keeper/ports/repositories.go`.
   - Implement repository persistence adapter in `internal/keeper/adapters/postgres/mcp_server_repository.go`.
3. **REFACTOR Phase**:
   - Refactor transport validation to use explicit error types instead of raw strings.
   - Decouple domain aggregate from GORM struct models if necessary to protect the hexagonal core.

## Acceptance Criteria

1. `mcp_server_test.go` and `mcp_server_repository_test.go` pass successfully.
2. The domain model, schema migrations, and repository persistence adapter are fully implemented and clean.
