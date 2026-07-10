# SPEC-FR-M6.5.5: MCP Client CRUD APIs + Per-Agent Tool Filter

| Field       | Value               |
|-------------|---------------------|
| ID          | SPEC-FR-M6.5.5      |
| Status      | DRAFT               |
| Milestone   | M6.5                |
| Component   | keeper              |
| Depends On  | SPEC-FR-M3.3        |
| Supersedes  | none                |

---

## Context

`MCPClient` is a first-class entity in the keeper domain model (`internal/keeper/domain/model/mcp_client.go`) with the following fields: `ID`, `TenantID`, `Name`, `Description`, `Transport` (stdio/sse), `Command`, `Args`, `Env`, `URL`, `AuthSecretRef`, and `Status` (active/suspended/inactive).

An agent can be attached to multiple MCP clients. For each attachment, the agent may optionally restrict which tools from that MCP server are exposed to the LLM via an `AllowedTools` filter. An agent already carries `MCPClients []MCPClientConfig{ClientID, AllowedTools, CustomEnv, CustomArgs}`.

When a hub agent has MCP clients attached but is deployed in hub mode, those clients are **not injected** into the hub runtime (the hub is routing-only for external actions). The hub-assignment validator emits a `warnings[]` field in the response for operator awareness.

This spec defines the full CRUD APIs for `MCPClient` and the per-agent attachment management endpoints.

---

## Specification

### 1. MCPClient CRUD

All endpoints are tenant-scoped. Tenant ID resolved from JWT claims or `X-Tenant-ID` header, propagated via `context.Context`.

| Method | Path                      | Description                             |
|--------|---------------------------|-----------------------------------------|
| GET    | /api/v1/mcp-clients       | List non-inactive MCP clients for tenant|
| POST   | /api/v1/mcp-clients       | Create a new MCP client                 |
| GET    | /api/v1/mcp-clients/{id}  | Get a single MCP client by ID           |
| PUT    | /api/v1/mcp-clients/{id}  | Update MCP client (metadata or status)  |
| DELETE | /api/v1/mcp-clients/{id}  | Soft-delete (set status=inactive)       |

**Transport rules (validated at creation/update):**
- If `transport=stdio`: `Command` is required; `URL` must be empty.
- If `transport=sse`: `URL` is required; `Command` must be empty.

**Secret Reference Handling:** The `AuthSecretRef` field stores a Kubernetes Secret name reference. The actual secret value is **never** returned in API responses. On any GET response, the `auth_secret_ref` field is **omitted** from the JSON output.

**Status lifecycle:** `active` / `suspended` / `inactive`. `inactive` clients do not appear in list responses. `suspended` clients are persisted but excluded from agent-runtime injection.

### 2. Agent Attachment Endpoints

| Method | Path                                                   | Description                                         |
|--------|--------------------------------------------------------|-----------------------------------------------------|
| POST   | /api/v1/agents/{id}/mcp-clients                        | Attach an MCP client to an agent                    |
| DELETE | /api/v1/agents/{id}/mcp-clients/{client_id}            | Detach an MCP client from an agent                  |
| GET    | /api/v1/agents/{id}/mcp-clients                        | List the agent's attached MCP clients with overrides|
| PUT    | /api/v1/agents/{id}/mcp-clients/{client_id}            | Update per-attachment overrides for a client        |

**POST body (attach):**
```json
{
  "client_id": "uuid",
  "allowed_tools": ["read_file", "list_dir"],
  "custom_env": {"TIMEOUT": "30"},
  "custom_args": ["--verbose"]
}
```

All override fields (`allowed_tools`, `custom_env`, `custom_args`) are optional. If `allowed_tools` is omitted or empty, all tools exposed by the MCP server are allowed for this agent.

**PUT body (update overrides):** Same shape as POST; all fields optional; only provided fields are updated on the attachment record.

**GET response item shape:**
```json
{
  "client_id": "uuid",
  "name": "filesystem-server",
  "transport": "stdio",
  "allowed_tools": ["read_file"],
  "custom_env": {},
  "custom_args": []
}
```

> `auth_secret_ref` is never included in attachment list responses.

### 3. AllowedTools Filter Semantics

- If `allowed_tools` is **empty or null** for an attachment: the agent's `ToolExecutor` receives all tools advertised by the MCP server at runtime.
- If `allowed_tools` is **non-empty**: only the listed tool names are passed to the `ToolExecutor`. Any tool name in `allowed_tools` that does not exist on the MCP server is silently ignored (logged at `WARN` level: `"allowed_tool {name} not found on MCP server {client_id}"`).
- The `AllowedTools` filter is the authoritative source; it is persisted in the `agent_mcp_clients` join table and delivered to the agent runtime via the Operator reconciler.

### 4. Hub Assignment Warning

When an agent with MCP clients attached is assigned the hub role (via the existing hub-assignment endpoint), the assignment **succeeds** (HTTP 200). The response body includes a `warnings` array:

```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
  "warnings": [
    "agent has MCP clients attached; MCP clients will NOT be injected in hub deployment mode (hub is routing-only)"
  ]
}
```

This is not an error. The attachment records are preserved. If the agent is later re-assigned to a non-hub role, MCP clients are injected normally.

### 5. Runtime Delivery

The per-agent MCP client attachment records (including `AllowedTools`, `CustomEnv`, `CustomArgs`) are the authoritative source delivered to the agent runtime. The Operator reconciler reads these records from Keeper and writes them into the agent pod's environment or configuration map (see SPEC-FR-M6.5.9). Only `active` MCP clients are included; `suspended` clients are excluded from the propagated config.

### 6. Caching

- Cache the per-agent MCP client list in Redis: key `agent-mcp-clients:{tenantID}:{agentID}`, TTL `60s` (Viper key: `cache.agent_mcp_clients_ttl`).
- Invalidate on: any agent MCP client attachment, detachment, or override update; any MCP client status change.

---

## Acceptance Criteria

1. **Create SSE client**: `POST /api/v1/mcp-clients` with `transport=sse` and a valid `url` succeeds and returns HTTP 201.
2. **Create stdio client without URL**: `POST /api/v1/mcp-clients` with `transport=stdio`, `command` set, and no `url` succeeds.
3. **Transport validation**: `POST` with `transport=sse` and no `url` returns HTTP 400. `POST` with `transport=stdio` and a `url` present returns HTTP 400.
4. **AllowedTools filter**: Attaching an MCP client with `allowed_tools=["read_file"]`; the GET attachment list returns `allowed_tools=["read_file"]` for that client.
5. **Hub assignment warning**: Assigning an agent with MCP clients as hub returns HTTP 200 with a non-empty `warnings[]` field mentioning MCP clients.
6. **Secret not exposed**: `GET /api/v1/mcp-clients/{id}` and `GET /api/v1/agents/{id}/mcp-clients` do not include `auth_secret_ref` in any response body.
7. **Detach removes from list**: After `DELETE /api/v1/agents/{id}/mcp-clients/{client_id}`, the client no longer appears in `GET /api/v1/agents/{id}/mcp-clients`.
8. **Override update**: `PUT /api/v1/agents/{id}/mcp-clients/{client_id}` with new `allowed_tools`; subsequent GET reflects the updated list.

---

## Test Plan

### Unit Tests

- `internal/keeper/domain/model/mcp_client_test.go`
  - `TestMCPClientValidate_SSE_RequiresURL` — missing URL with sse transport fails.
  - `TestMCPClientValidate_Stdio_RequiresCommand` — missing command with stdio transport fails.
  - `TestMCPClientValidate_SSE_NoCommand_Succeeds`.
  - `TestMCPClientValidate_Stdio_NoURL_Succeeds`.
  - `TestMCPClientValidate_InvalidTransport` — unknown transport returns error.
  - `TestMCPClientValidate_StatusValues` — active/suspended/inactive pass; unknown fails.

### Unit Tests — Hub Warning Logic

- `internal/keeper/application/service/agent_service_test.go`
  - `TestAssignHubRole_WithMCPClients_Returns200WithWarning`.
  - `TestAssignHubRole_NoMCPClients_Returns200NoWarning`.

### Integration Tests (Gin TestMode + httptest)

- `internal/keeper/adapters/inbound/http/mcp_client_handler_test.go`
  - `TestCreateMCPClient_SSE_Success` — 201, no `auth_secret_ref` in response.
  - `TestCreateMCPClient_SSEMissingURL_Returns400`.
  - `TestCreateMCPClient_StdioMissingCommand_Returns400`.
  - `TestAttachMCPClientToAgent_AllowedTools` — verifies `allowed_tools` stored correctly.
  - `TestGetAgentMCPClients_SecretNotExposed` — verifies `auth_secret_ref` absent.
  - `TestUpdateAttachmentOverrides` — PUT updates `allowed_tools`; GET reflects change.
  - `TestDetachMCPClient_RemovedFromList`.
  - `TestHubAssignment_WithMCPClients_WarningPresent`.
  - `TestCacheInvalidationOnDetach` — mock Redis; DEL called on detach.

---

## API Contract

### POST /api/v1/mcp-clients — SSE Transport

**Request Body:**
```json
{
  "name": "filesystem-server",
  "description": "MCP server exposing filesystem tools via SSE",
  "transport": "sse",
  "url": "https://mcp.internal/filesystem/sse",
  "auth_secret_ref": "k8s-secret-mcp-filesystem"
}
```

**Response 201:**
```json
{
  "id": "883b0700-e29b-41d4-a716-446655440003",
  "tenant_id": "tenant-abc",
  "name": "filesystem-server",
  "description": "MCP server exposing filesystem tools via SSE",
  "transport": "sse",
  "url": "https://mcp.internal/filesystem/sse",
  "status": "active",
  "created_at": "2026-07-10T08:00:00Z",
  "updated_at": "2026-07-10T08:00:00Z"
}
```

> Note: `auth_secret_ref` is intentionally absent from the response.

### POST /api/v1/mcp-clients — Stdio Transport

**Request Body:**
```json
{
  "name": "git-mcp",
  "description": "MCP server for git operations",
  "transport": "stdio",
  "command": "/usr/local/bin/git-mcp",
  "args": ["--repo", "/workspace"],
  "env": {"GIT_AUTHOR_NAME": "agent"}
}
```

### POST /api/v1/agents/{id}/mcp-clients

**Request Body:**
```json
{
  "client_id": "883b0700-e29b-41d4-a716-446655440003",
  "allowed_tools": ["read_file", "list_dir"],
  "custom_env": {"BASE_PATH": "/data"},
  "custom_args": []
}
```

**Response 200:** `{"message": "mcp client attached to agent"}`

### GET /api/v1/agents/{id}/mcp-clients

**Response 200:**
```json
{
  "agent_id": "agent-uuid",
  "mcp_clients": [
    {
      "client_id": "883b0700-e29b-41d4-a716-446655440003",
      "name": "filesystem-server",
      "transport": "sse",
      "allowed_tools": ["read_file", "list_dir"],
      "custom_env": {"BASE_PATH": "/data"},
      "custom_args": []
    }
  ],
  "total": 1
}
```

### PUT /api/v1/agents/{id}/mcp-clients/{client_id}

**Request Body:**
```json
{
  "allowed_tools": ["read_file"],
  "custom_env": {}
}
```

**Response 200:** `{"message": "mcp client attachment updated"}`

### Hub Assignment Response with Warning

**POST /api/v1/agents/{id}/role — Response 200:**
```json
{
  "agent_id": "agent-uuid",
  "role": "hub",
  "warnings": [
    "agent has MCP clients attached; MCP clients will NOT be injected in hub deployment mode (hub is routing-only)"
  ]
}
```

### Error Response (all endpoints)

```json
{ "error": "descriptive error message" }
```

### OpenAPI Tag

All MCP client operations carry the tag: `infrastructure/mcp-clients`

---

## Files Affected

| File                                                                       | Action                                              |
|----------------------------------------------------------------------------|-----------------------------------------------------|
| `internal/keeper/domain/model/mcp_client.go`                               | EXISTING — CRUD spec formalises the model; add transport validation method |
| `internal/keeper/domain/model/agent.go`                                    | EXISTING — `MCPClients []MCPClientConfig` already present; verify `AllowedTools` field in `MCPClientConfig` |
| `internal/keeper/application/ports/inbound/mcp_client_service.go`          | NEW — inbound port interface                        |
| `internal/keeper/application/ports/outbound/mcp_client_repository.go`      | NEW — outbound port interface                       |
| `internal/keeper/application/service/mcp_client_service.go`                | NEW — use case orchestration                        |
| `internal/keeper/adapters/inbound/http/mcp_client_handler.go`              | NEW — Gin HTTP handler                              |
| `internal/keeper/adapters/outbound/postgres/mcp_client_repository.go`      | NEW — pgx adapter                                   |
| `internal/keeper/adapters/outbound/redis/mcp_client_cache.go`              | NEW — Redis cache adapter                           |
| `db/migrations/<timestamp>_create_mcp_clients.sql`                         | NEW — mcp_clients, agent_mcp_clients tables         |
