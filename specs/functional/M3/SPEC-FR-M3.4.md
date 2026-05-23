# SPEC-FR-M3.4: Prompt Collections & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.4                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3                                |
| Supersedes    | none                                        |

## Context

Prompts define the behavior, persona, and instructions for an agent. To keep prompts structured and reusable, keeper manages a catalog of **Prompt Templates** and groups them into **Prompt Collections**. These collections represent the suite of templates (e.g., System Prompt, Greeting Prompt, Error Recovery Prompt) associated with a specific agent profile. By resolving prompts dynamically at runtime, keeper ensures agent containers are injected with the correct instructions while preserving clean versioning.

## Specification

1. The system MUST define a `PromptTemplate` entity representing a single parameterized prompt template:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique name within version history)
   - `content`: Text (Template body with placeholders, e.g. `You are {{.Name}}...`)
   - `role`: String (Enum: `system`, `user`, `assistant`)
   - `version`: Integer (Auto-incrementing, immutable)
   - `status`: Enum (`draft`, `active`, `archived`)
   - `created_at`: Timestamp
2. The system MUST define a `PromptCollection` aggregate in the keeper domain layer representing a collection of prompt templates:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique, required)
   - `description`: String (Optional)
   - `templates`: List of associated `PromptTemplate` identifiers/references
   - `created_at`: Timestamp
   - `updated_at`: Timestamp
3. The keeper MUST expose CRUD REST endpoints to manage both prompt templates and prompt collections:
   - `GET/POST /api/v1/prompts`: List/Create a prompt template.
   - `GET/PUT/DELETE /api/v1/prompts/{id}`: View, update (creates a new version), or delete prompt templates.
   - `GET/POST /api/v1/prompt-collections`: List/Create prompt collections.
   - `GET/PUT/DELETE /api/v1/prompt-collections/{id}`: View, update, or delete prompt collections.
4. Prompt updates MUST create a new version of the template (immutable versions) to satisfy history tracking.
5. The domain layer MUST NOT import adapter or application packages (per `SPEC-NFR-HEXAGONAL`).
6. Input validation MUST use Gin binding tags (per `SPEC-NFR-HTTP`).
7. Prompt resolution MUST select the latest `active` version of any template assigned to a collection.

## Acceptance Criteria

1. **Domain Model**:
   - `PromptTemplate` and `PromptCollection` aggregates are fully defined in the keeper domain layer.
   - Versioning logic ensures modifying a prompt template results in a new immutable template version.
2. **API Endpoint Integration**:
   - CRUD API successfully manages collections and templates with structured validation.
   - Prompt templates support version filtering when retrieved.
3. **Hexagonal Boundaries**:
   - No external packages or adapter details leaking into domain logic.

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Validate version auto-incrementing logic when creating new revisions.
   - Validate placeholder interpolation or syntax checking if applicable.
2. **Integration Tests**:
   - REST API controller tests using the Gin HTTP framework checking version increments and collection association.

## Files Affected

- `internal/keeper/domain/prompt.go` [NEW] — Defines the `PromptTemplate` and `PromptCollection` structures.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares repository ports for prompts and collections.
- `internal/keeper/adapters/http/prompt_handlers.go` [NEW] — Implements API controllers for prompt management.
- `internal/keeper/bootstrap.go` [MODIFY] — Binds prompt routes onto the Gin engine.
