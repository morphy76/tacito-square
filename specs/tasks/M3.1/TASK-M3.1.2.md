# TASK-M3.1.2: LLM Provider Bindings HTTP API Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.1.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M3.1                                |
| Depends On    | TASK-M3.1.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementation for LLM Provider Bindings. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create HTTP integration tests in `internal/keeper/adapters/http/llm_binding_handlers_test.go` verifying:
     - `POST /api/v1/llm-bindings` creates a binding and returns 201.
     - `POST /api/v1/llm-bindings` rejects missing or malformed inputs (returns 400 Bad Request with JSON error).
     - `GET /api/v1/llm-bindings` returns all active bindings list.
     - `GET /api/v1/llm-bindings/{id}` returns the binding or 404.
     - `PUT /api/v1/llm-bindings/{id}` updates attributes successfully.
     - `DELETE /api/v1/llm-bindings/{id}` deletes resource.
2. **GREEN Phase**:
   - Define Gin binding tags on input request models to automatically validate request schemas.
   - Implement the HTTP controller handlers in `internal/keeper/adapters/http/llm_binding_handlers.go`.
   - Wire the routes onto the Gin engine inside `internal/keeper/bootstrap.go`.
3. **REFACTOR Phase**:
   - Refactor handler response helper signatures to enforce consistency in standard JSON error formats (per `SPEC-NFR-HTTP`).
   - Clean up mapping structures, error checking, and HTTP tests configuration.

## Acceptance Criteria

1. `llm_binding_handlers_test.go` HTTP controller integration tests pass successfully with race detector enabled.
2. Gin endpoints successfully validate requests, persist records via repository ports, and return standard JSON formats.
