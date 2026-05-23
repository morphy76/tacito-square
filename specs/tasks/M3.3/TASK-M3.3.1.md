# TASK-M3.3.1: Skill Collections Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.3.1                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.3                                |
| Depends On    | TASK-M3.2.2                                 |

## Description

Define the domain model, tool whitelisting and blacklisting validation rules, repository interfaces, and GORM persistence adapter for Skills. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/domain/skill_test.go` with domain tests:
     - `TestSkill_ToolAuthorization` — verifies that whitelists (`allowed_tools`) and blacklists (`denied_tools`) correctly filter and authorize tool names dynamically.
   - Create GORM persistence integration tests in `internal/keeper/adapters/postgres/skill_repository_test.go` checking Skill creation and relational links to MCP server IDs.
2. **GREEN Phase**:
   - Create `internal/keeper/domain/skill.go` containing the `Skill` aggregate struct and whitelisting/blacklisting logical helpers.
   - Define database schema migration in `deploy/postgres/migrations/` (defining `skills` and relational join tables for associated `mcp_servers`).
   - Register repository port in `internal/keeper/ports/repositories.go` declaring repository port interfaces.
   - Implement repository persistence adapter in `internal/keeper/adapters/postgres/skill_repository.go`.
3. **REFACTOR Phase**:
   - Optimize database preload queries for associated MCP servers.
   - Ensure GORM-specific models are decoupled from the pure domain aggregate models.

## Acceptance Criteria

1. `skill_test.go` and `skill_repository_test.go` pass successfully.
2. The domain model, schema migrations, and repository persistence adapter are fully implemented and free of hexagonal leaks.
