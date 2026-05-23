# TASK-M3.7.2: Agent-Community Assignment HTTP API & Observability Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.7.2                                 |
| Status        | CLOSED                                      |
| Spec          | SPEC-FR-M3.7                                |
| Depends On    | TASK-M3.7.1                                 |

## Description

Implement the HTTP handlers, Gin routes, payload validations, and cross-tenant guard middleware for assigning and unassigning agent templates to communities. Focus heavily on logging, distributed tracing, and correlation requirements specified in `SPEC-NFR-LOG` and `SPEC-NFR-OBSERVABILITY`.

## Work Items

1. **RED Phase**:
   - Create HTTP unit tests in `internal/keeper/adapters/http/assignment_handlers_test.go` (using mocked repositories) to verify:
     - `POST /api/v1/communities/:community_id/agents/:agent_id` returns `200 OK` (or `204 No Content`) and successfully triggers database updates.
     - Attempting to assign an agent that is already assigned returns a `409 Conflict` response with a structured JSON error.
     - Attempting to assign cross-tenant entities (e.g. `community` of Tenant A to `agent` of Tenant B) returns `404 Not Found` to prevent data leakage.
     - `DELETE /api/v1/communities/:community_id/agents/:agent_id` returns `204 No Content` and clears the assignment in the DB.
     - Requests without valid multitenancy context headers return `401 Unauthorized`.
   - Write test assertions to verify that database client operations execute inside the parent request trace span.

2. **GREEN Phase**:
   - Register route endpoints under `/api/v1` in `internal/keeper/bootstrap.go`:
     - `POST /communities/:community_id/agents/:agent_id`
     - `DELETE /communities/:community_id/agents/:agent_id`
   - Implement the handler functions in a controller (such as `internal/keeper/adapters/http/community_handlers.go` or a new handler file).
   - Ensure the handler resolves both community and agent within the context tenant and validates status invariants.
   - **Logging and Tracing Instrumentation**:
     - Inject OpenTelemetry child spans for assignment and unassignment operations (correlated to parent HTTP request span per `SPEC-NFR-OBSERVABILITY`).
     - Log structured JSON logs using `zerolog` (per `SPEC-NFR-LOG`) on every lifecycle transition, capturing standard fields (`trace_id`, `span_id`, `tenant_id`, `agent_id`, `community_id`, `transition`).

3. **REFACTOR Phase**:
   - Refactor handler validation rules to leverage Gin binding conventions.
   - Clean up tracing span definitions using defer semantics to prevent leakages under error/panic flows.

## Acceptance Criteria

1. All HTTP handlers unit tests pass cleanly using mocked repository configurations.
2. Endpoint validations return correct HTTP status codes (`200 OK`, `400 Bad Request`, `404 Not Found`, `409 Conflict`).
3. Transition logs are strictly written in structured JSON and automatically carry active trace/span correlation IDs.
