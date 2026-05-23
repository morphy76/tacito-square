# TASK-M3.6.1: Community Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.6.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.6                                |
| Depends On    | none                                        |

## Description

Define the `Community` aggregate domain model, topology and status enums, validation business rules, repository interfaces, and GORM database persistence adapter structures. Modify the `Agent` aggregate and database migration schemas to establish a single-community relationship via a nullable foreign key constraint. Follow a strict TDD lifecycle within this domain and interface boundary, in compliance with `SPEC-NFR-HEXAGONAL` and `SPEC-NFR-MULTITENANCY`.

## Work Items

1. **RED Phase**:
   - Create domain unit tests in `internal/keeper/domain/community_test.go` verifying:
     - Basic validations (mandatory fields: `id`, `name`, `tenant_id`, `topology`, `status`).
     - Validation invariants for the supported topologies (e.g., validation fail for unsupported values).
     - Lifecycle status transition constraints (valid statuses: `created`, `active`, `suspended`, `terminated`).
   - Extend/update domain unit tests in `internal/keeper/domain/agent_test.go` to cover the validation of the nullable `community_id *uuid.UUID` field.

2. **GREEN Phase**:
   - Create `internal/keeper/domain/community.go` defining the `Community` aggregate, status, topology enums, and validation logic.
   - Modify the `Agent` structure in `internal/keeper/domain/agent.go` to add the `CommunityID *uuid.UUID` field and update `Validate()` validation invariants.
   - Declare the `CommunityRepository` interface in `internal/keeper/ports/repositories.go`.
   - Add database Goose migrations under a new SQL file (e.g., in the migration directory) defining:
     - The `communities` table with non-nullable `tenant_id` and unique composite key `(tenant_id, name)`.
     - Altering the `agents` table to add the `community_id` column with index `idx_agents_community` and foreign key constraint referencing `communities(id) ON DELETE RESTRICT`.
   - Implement the `postgres.CommunityRepository` adapter in `internal/keeper/adapters/postgres/community_repository.go`.

3. **REFACTOR Phase**:
   - Refactor domain and validation structures to optimize performance.
   - Audit code imports to guarantee the domain entities are strictly decoupled and do not import framework packages (Gin, GORM, database driver packages) as required by `SPEC-NFR-HEXAGONAL`.

## Acceptance Criteria

1. All domain unit tests in `community_test.go` and `agent_test.go` pass successfully.
2. The `CommunityRepository` port and `postgres.CommunityRepository` adapter structures are compiled and declared correctly without any hexagonal leaks.
3. The Goose migration is correct and sets up GORM-compatible mappings.
