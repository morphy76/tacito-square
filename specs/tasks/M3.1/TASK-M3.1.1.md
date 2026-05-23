# TASK-M3.1.1: LLM Provider Bindings Domain & Repository Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.1.1                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.1                                |
| Depends On    | none                                        |

## Description

Define the domain model, business rules, persistence repository interfaces, and Postgres/GORM adapter implementation for LLM Provider Bindings. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/domain/llm_binding_test.go` verifying:
     - Model instantiation and validation constraints (uniqueness of name, mandatory fields: `name`, `provider`, `api_base_url`, `default_model`).
     - Allowed provider types enum validation (`openai`, `anthropic`, `groq`, `ollama`, `custom`).
   - Create database integration tests in `internal/keeper/adapters/postgres/llm_binding_repository_test.go` verifying GORM read/write persistence.
2. **GREEN Phase**:
   - Create `internal/keeper/domain/llm_binding.go` with domain definition and validation logic.
   - Declare GORM database schema and goose migrations in `deploy/postgres/migrations/` (defining LLM bindings table with index constraints).
   - Define repository ports in `internal/keeper/ports/repositories.go`.
   - Implement the repository persistence adapter in `internal/keeper/adapters/postgres/llm_binding_repository.go`.
3. **REFACTOR Phase**:
   - Refactor repository query efficiency and connection pooling error handling.
   - Ensure the domain model remains completely decoupled from external frameworks.

## Acceptance Criteria

1. `llm_binding_test.go` and `llm_binding_repository_test.go` pass successfully.
2. The domain model, migration scripts, and repository adapter are fully implemented and free of hexagonal leaks.
