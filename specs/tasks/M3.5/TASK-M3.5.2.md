# TASK-M3.5.2: Agent HTTP API Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.5.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.5                                |
| Depends On    | TASK-M3.5.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementation for Agent templates CRUD operations. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create HTTP integration tests in `internal/keeper/adapters/http/agent_handlers_test.go` verifying:
     - `POST /api/v1/agents` successfully creates an Agent template and returns 201 Created.
     - `POST /api/v1/agents` rejects missing or malformed inputs and returns 400 Bad Request with JSON error response.
     - `GET /api/v1/agents` returns all active agent templates list for the context tenant.
     - `GET /api/v1/agents/{id}` returns the specific agent template or 404.
     - `PUT /api/v1/agents/{id}` updates attributes successfully.
     - `DELETE /api/v1/agents/{id}` deletes/soft deletes the agent template.
     - Verify strict multi-tenant context enforcement across all routes.
   - Set up contract test assertions verifying `GET /openapi.json` correctly exposes the new `/agents` schemas and routes.
2. **GREEN Phase**:
   - Define Gin binding tags on input request models to automatically validate request schemas.
   - Implement the HTTP controller handlers in `internal/keeper/adapters/http/agent_handlers.go`.
   - Wire the routes onto the Gin engine inside `internal/keeper/bootstrap.go`.
   - Update `internal/keeper/openapi.json` to define all Agent routes (`/agents`, `/agents/{id}`), request/response schemas, and error responses.
   - Declare the stable `agent-config/agents` tag in the top-level `tags` array of the OpenAPI spec with a descriptive context summary, and ensure each operation carries exactly this tag (as required by `SPEC-NFR-OPENAPI`).
3. **REFACTOR Phase**:
   - Refactor response helpers to enforce consistency in standard JSON error formats.
   - Clean up mapping structures, error checking, and HTTP tests configuration.
   - Audit the new OpenAPI path definitions to ensure 100% compliance with `SPEC-NFR-OPENAPI` conventions.

## Acceptance Criteria

1. `agent_handlers_test.go` integration tests pass successfully.
2. Gin endpoints successfully validate requests, persist records via repository ports, and return standard JSON formats.
3. The embedded `internal/keeper/openapi.json` contains complete, valid OpenAPI 3.x definitions for the new Agent endpoints.
4. The OpenAPI changes are validated successfully by OpenAPI contract tests, and adhere perfectly to the single `domain/subdomain` tag constraint (`agent-config/agents`) of `SPEC-NFR-OPENAPI`.

