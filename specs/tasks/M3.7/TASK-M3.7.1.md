# TASK-M3.7.1: Agent-Community Assignment Repository & Domain Logic

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.7.1                                 |
| Status        | CLOSED                                      |
| Spec          | SPEC-FR-M3.7                                |
| Depends On    | none                                        |

## Description

Define and implement the core domain invariants, status transitions, and persistence repository logic for assigning and unassigning agent templates to communities. Implement these changes in a transaction-safe manner in GORM/PostgreSQL, ensuring the database state correctly enforces the "One-Community Constraint" and transitions status properly. Follow a strict TDD lifecycle within this domain and interface boundary, in compliance with `SPEC-NFR-HEXAGONAL` and `SPEC-NFR-MULTITENANCY`.

## Work Items

1. **RED Phase**:
   - Create domain unit tests in `internal/keeper/domain/agent_test.go` (or a dedicated assignment test file) verifying:
     - The transitioning of Agent status from `AgentStatusDefined` to `AgentStatusAssigned` when community assignment occurs.
     - Validating that an agent already assigned to a community cannot be assigned to another community without explicit validation failure.
     - The transitioning of Agent status back to `AgentStatusDefined` when the agent is unassigned (the community ID becomes null).
   - **Integration Tests (Testcontainers)**:
     - Create database repository integration tests (e.g. in `internal/keeper/adapters/postgres/agent_repository_test.go`) verifying:
       - Successful community assignment updates `community_id` in the database and transitions agent status properly.
       - Relational constraint violations and cross-tenant guard assertions fail with proper transaction rollbacks.
     - **Test Isolation**: Strictly segregate these integration tests from the core package unit tests by placing the standard `//go:build integration` build tag flag at the very top of the test file, ensuring they are only run when explicitly invoking `-tags=integration`.

2. **GREEN Phase**:
   - Update `internal/keeper/domain/agent.go` to include `community_id` in validation rules and support the transition validations.
   - Modify the `AgentRepository` port in `internal/keeper/application/ports/outbound/repositories.go` (or implement dynamic assignment repository ports) to declare:
     - `AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error`
     - `UnassignFromCommunity(ctx context.Context, agentID uuid.UUID) error`
   - Implement these methods inside the GORM/PostgreSQL adapter `internal/keeper/adapters/postgres/agent_repository.go`. Wrap operations in transaction boundaries where appropriate.
   - Create direct Goose migrations to enforce relational foreign keys and index performance:
     - Ensure `agents.community_id` has a proper index for fast reverse lookups (listing agents in a community).

3. **REFACTOR Phase**:
   - Optimize GORM queries to avoid redundant reads.
   - Validate architectural boundaries using `go vet`, ensuring no infrastructure/adapter leaks into the core domain or port packages (compliant with `SPEC-NFR-HEXAGONAL`).

## Acceptance Criteria

1. All domain unit tests and repository integration tests pass successfully.
2. Integration tests are strictly isolated via the Go build tag flag `//go:build integration` at the very top of the file, keeping them completely segregated from the standard, fast unit tests.
3. The `AgentRepository` port and GORM adapter declare and implement assignment methods cleanly.
4. High-performance indexes and foreign key constraints are established in schema migrations.
