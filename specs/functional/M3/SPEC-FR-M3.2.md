# SPEC-FR-M3.2: MCP Servers & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.2                                |
| Status        | DRAFT                                       |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3                                |
| Supersedes    | none                                        |

## Context

To perform actions and interact with external development environments, filesystems, databases, or third-party APIs, agents utilize **Model Context Protocol (MCP)** clients. The keeper service manages a registry of **MCP Server Configurations**. These configurations outline how active agents should spawn and connect to MCP servers (either via Local Command Exec/stdio or via Remote HTTP/SSE endpoints) at runtime, separating tool execution setup from the core agent behavior.

## Specification

1. The system MUST define an `MCPServer` aggregate in the keeper domain layer representing an MCP server connection profile with fields:
   - `id`: UUID (Primary Key)
   - `name`: String (Unique, required - e.g., `postgresql-mcp`, `github-mcp`)
   - `description`: String (Optional)
   - `transport`: String (Enum: `stdio`, `sse`)
   - `command`: String (Required if transport is `stdio` - path to executable)
   - `args`: List of Strings (Optional, arguments passed to executable)
   - `env`: Map of String to String (Optional, environment variables for execution)
   - `url`: String (Required if transport is `sse` - HTTP/SSE endpoint URL)
   - `auth_secret_ref`: String (Optional, Kubernetes secret reference containing access tokens/headers)
   - `status`: Enum (`active`, `suspended`, `inactive`)
   - `created_at`: Timestamp
   - `updated_at`: Timestamp
2. The keeper MUST expose CRUD REST endpoints to manage MCP server profiles:
   - `POST /api/v1/mcp-servers`: Register a new MCP server profile.
   - `GET /api/v1/mcp-servers`: List registered MCP server profiles.
   - `GET /api/v1/mcp-servers/{id}`: Retrieve a specific MCP server profile.
   - `PUT /api/v1/mcp-servers/{id}`: Update an MCP server profile.
   - `DELETE /api/v1/mcp-servers/{id}`: Unregister an MCP server profile.
3. The domain layer MUST NOT import adapter or application packages (per `SPEC-NFR-HEXAGONAL`).
4. Input validation MUST use Gin binding tags (per `SPEC-NFR-HTTP`).
5. Sensitive environment variables or authorization tokens MUST be stored/referenced via Kubernetes secrets (`auth_secret_ref`) instead of plain text in the database.

## Acceptance Criteria

1. **Domain Model**:
   - `MCPServer` aggregate correctly differentiates transport validation (e.g., command is required for `stdio`, URL is required for `sse`).
2. **API Endpoint Integration**:
   - Valid configurations are successfully stored and retrieved.
   - Configurations violating transport constraints (e.g. `sse` missing a URL) are rejected with a 400 Bad Request standard JSON error.
3. **Hexagonal Boundaries**:
   - Strict separation maintained in the domain package.

## Test Plan

### Automated Tests
1. **Unit Tests**:
   - Test transport-specific struct validation rules.
2. **Integration Tests**:
   - Mock API testing via Gin covering registration, listing, updating, and unregistering MCP server configurations.

## Files Affected

- `internal/keeper/domain/mcp_server.go` [NEW] — Defines the `MCPServer` model.
- `internal/keeper/ports/repositories.go` [MODIFY] — Declares repository ports for MCP servers.
- `internal/keeper/adapters/http/mcp_server_handlers.go` [NEW] — Implements API controllers for MCP server profiles.
- `internal/keeper/bootstrap.go` [MODIFY] — Binds MCP server routes onto the Gin engine.
