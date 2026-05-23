# TASK-M3.6.2: Community HTTP API Boundary & OpenAPI Contract

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.6.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.6                                |
| Depends On    | TASK-M3.6.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementations for Community templates CRUD operations. Follow a strict TDD lifecycle within this boundary, using mocked repository interfaces to isolate the HTTP boundary. Update the OpenAPI specifications and validate integration with OpenAPI contract tests in compliance with `SPEC-NFR-HTTP` and `SPEC-NFR-OPENAPI`.

## Work Items

1. **RED Phase**:
   - Create HTTP unit tests in `internal/keeper/adapters/http/community_handlers_test.go` utilizing a **mocked repository implementation** (simulating `CommunityRepository` ports) to verify:
     - `POST /api/v1/communities` successfully creates a Community template and returns `201 Created` with a matching payload.
     - `POST /api/v1/communities` rejects missing or malformed inputs (e.g. empty names, invalid topologies) and returns `400 Bad Request` with a standard JSON error response.
     - `GET /api/v1/communities` returns the list of active community configurations for the context tenant.
     - `GET /api/v1/communities/:id` returns the community detail or `404 Not Found` if missing.
     - `PUT /api/v1/communities/:id` updates community specifications successfully.
     - `DELETE /api/v1/communities/:id` deletes a community successfully if no agents are assigned to it, or returns an error response (e.g., `409 Conflict`) if the delete constraint is violated.
     - Verify strict multi-tenant context enforcement across all routes (requests without headers return `401 Unauthorized`).
   - Set up contract test assertions verifying `GET /openapi.json` correctly exposes the new `/communities` schemas and routes.

2. **GREEN Phase**:
   - Define Gin binding tags on input request models to automatically validate request schemas.
   - Implement the HTTP controller handlers in `internal/keeper/adapters/http/community_handlers.go`.
   - Wire the routes onto the Gin engine inside `internal/keeper/bootstrap.go` under the `/api/v1` group (ensuring the tenant resolution middleware protects these endpoints).
   - Update `internal/keeper/openapi.json` to define all Community routes (`/communities`, `/communities/{id}`), request/response schemas, and error responses.
   - Declare the stable `community/communities` tag in the top-level `tags` array of the OpenAPI spec with a descriptive context summary, and ensure each operation carries exactly this tag (as required by `SPEC-NFR-OPENAPI`).

3. **REFACTOR Phase**:
   - Refactor handler response models to use standard error formatting from the shared packages.
   - Audit the new paths and models defined in `internal/keeper/openapi.json` to ensure 100% compliance with `SPEC-NFR-OPENAPI` rules.

## Acceptance Criteria

1. All HTTP handlers unit tests in `community_handlers_test.go` (running without requiring Docker or a database connection) pass successfully.
2. Gin endpoints successfully validate requests, persist records via mocked repository ports, and return standard JSON formats.
3. The embedded `internal/keeper/openapi.json` contains complete, valid OpenAPI 3.x definitions for the new Community endpoints.
4. The OpenAPI changes are validated successfully by OpenAPI contract tests, and adhere perfectly to the single `domain/subdomain` tag constraint (`community/communities`) of `SPEC-NFR-OPENAPI`.
