# TASK-M3.6.3: Community Integration Tests (Testcontainers)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.6.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.6                                |
| Depends On    | TASK-M3.6.1                                 |

## Description

Validate the database persistence schemas, GORM mappings, and tenant constraints of the Community repository against a live, isolated PostgreSQL container running in Docker. Leverage the existing `testcontainers-go` harness in `internal/keeper/adapters/postgres/main_test.go` to automatically spin up a database instance and execute versioned Goose migrations, ensuring complete environment parity with production. Follow a strict TDD lifecycle within this containerized integration boundary.

## Work Items

1. **RED Phase**:
   - Write high-fidelity database assertions in `internal/keeper/adapters/postgres/community_repository_test.go` verifying:
     - **Composite Unique Index**: Verifying that a duplicate `(tenant_id, name)` insert throws a driver-level unique key violation error.
     - **Foreign Key Cascade Integrity**: Verifying that attempting to delete a community with active assigned agents throws a driver-level foreign key constraint violation (due to the `ON DELETE RESTRICT` rule).
     - **JSONB Configuration Mapping**: Verifying that complex nested JSON structures in the community configuration are serialized and deserialized correctly to and from the PostgreSQL `JSONB` column.
     - **Multi-Tenant Scoping**: Verifying that querying lists or searching by name resolves strictly within the tenant context injected in the connection, returning no rows for other tenants' records.

2. **GREEN Phase**:
   - Execute the test suite via the command:
     ```bash
     make test-integration
     ```
   - Ensure the `main_test.go` migration parser correctly parses the newly added Goose migration scripts (containing table creations, columns, foreign keys, and indexes) and applies them to the test container without syntax or connection errors.
   - Address GORM struct tags and repository mapping bugs until all integration assertions pass against the container.

3. **REFACTOR Phase**:
   - Refactor SQL queries or GORM statements in the repository adapter to optimize performance (e.g., proper index usage, avoiding unnecessary joins).
   - Ensure all database cleanup hooks (e.g., deleting test records) in the integration tests run cleanly after each test case to prevent test pollution in parallel runs.

## Acceptance Criteria

1. Running `make test-integration` starts the PostgreSQL test container successfully, applies all migrations, and all `community_repository_test.go` test cases pass.
2. Unique composite constraints and foreign key constraints are verified at the true database engine layer.
3. Complex `JSONB` metadata fields map cleanly without data truncation.
