# TASK-M3.4.1: Prompt Collections Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.4.1                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.4                                |
| Depends On    | none                                        |

## Description

Define the domain models, GORM persistent schemas, auto-incrementing immutable version control validation, and GORM persistence adapter for Prompt Templates and Prompt Collections. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/domain/prompt_test.go` with domain tests:
     - `TestPromptTemplate_ImmutableVersions` — verifies that modifying or updating a prompt template creates an entirely new record in GORM database with an incremented version number, preserving previous versions.
   - Create GORM persistence integration tests in `internal/keeper/adapters/postgres/prompt_repository_test.go` checking prompt listing and collection-to-template linking.
2. **GREEN Phase**:
   - Create `internal/keeper/domain/prompt.go` containing `PromptTemplate` and `PromptCollection` aggregate structs and version incrementing domain logic.
   - Define database schema migration in `deploy/postgres/migrations/` (creating `prompt_templates`, `prompt_collections`, and join tables).
   - Register repository port in `internal/keeper/ports/repositories.go`.
   - Implement repository persistence adapter in `internal/keeper/adapters/postgres/prompt_repository.go`.
3. **REFACTOR Phase**:
   - Refactor repository query lookup to optimize prompt selection (specifically resolving the latest active version of a template assigned to a collection).
   - Verify decoupling parameters.

## Acceptance Criteria

1. `prompt_test.go` and `prompt_repository_test.go` pass successfully.
2. The domain models, schema migrations, and repository persistence adapter are fully implemented and clean.
