# TASK-M3.4.2: Prompt Collections HTTP API Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.4.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.4                                |
| Depends On    | TASK-M3.4.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementation for Prompt Templates and Prompt Collections. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create HTTP integration tests in `internal/keeper/adapters/http/prompt_handlers_test.go` verifying:
     - `POST /api/v1/prompts` registers a prompt template and returns 201.
     - `GET /api/v1/prompts` retrieves prompt templates.
     - `GET /api/v1/prompts/{id}` retrieves single template.
     - `PUT /api/v1/prompts/{id}` creates a new immutable version and returns 201/200.
     - `POST /api/v1/prompt-collections` creates a prompt collection.
     - `GET /api/v1/prompt-collections/{id}` fetches collection.
     - `DELETE /api/v1/prompt-collections/{id}` deletes collection.
2. **GREEN Phase**:
   - Add schema validation tags on request parameters.
   - Implement the HTTP controller handlers in `internal/keeper/adapters/http/prompt_handlers.go`.
   - Setup route mapping configuration in `internal/keeper/bootstrap.go`.
3. **REFACTOR Phase**:
   - Clean up handler version filtering variables to protect against unauthorized SQL injections.
   - Refactor standard response templates.

## Acceptance Criteria

1. `prompt_handlers_test.go` HTTP controller integration tests pass successfully with race detector enabled.
2. Gin endpoints successfully validate payload fields, enforce immutable versioning via repository ports, and return standard JSON formats.
