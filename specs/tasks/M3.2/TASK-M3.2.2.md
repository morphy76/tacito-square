# TASK-M3.2.2: MCP Servers HTTP API Boundary

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.2.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M3.2                                |
| Depends On    | TASK-M3.2.1                                 |

## Description

Define the REST API request/response contracts, Gin routes, payload validation binding tags, and HTTP handler implementation for MCP Servers. Follow a strict TDD lifecycle within this boundary.

## Work Items

1. **RED Phase**:
   - Create HTTP integration tests in `internal/keeper/adapters/http/mcp_server_handlers_test.go` verifying:
     - `POST /api/v1/mcp-servers` registers a server profile and returns 201.
     - `POST /api/v1/mcp-servers` rejects payload violating transport rules (e.g. `sse` missing a URL) and returns 400 Bad Request.
     - `GET /api/v1/mcp-servers` retrieves listing.
     - `GET /api/v1/mcp-servers/{id}` retrieves single profile.
     - `PUT /api/v1/mcp-servers/{id}` updates fields successfully.
     - `DELETE /api/v1/mcp-servers/{id}` unregisters profile.
2. **GREEN Phase**:
   - Add schema validation Gin binding tags on request DTOs.
   - Implement the HTTP controller handlers in `internal/keeper/adapters/http/mcp_server_handlers.go`.
   - Wire the routes onto the Gin engine in `internal/keeper/bootstrap.go`.
3. **REFACTOR Phase**:
   - Refactor transport validation handling within controllers to return helpful, structured parameter error objects in the JSON payload.
   - Clean up handler tests bootstrapping.

## Acceptance Criteria

1. `mcp_server_handlers_test.go` HTTP controller integration tests pass successfully with race detector enabled.
2. Gin endpoints successfully validate transport constraints, persist profiles via repository ports, and return standard JSON formats.
