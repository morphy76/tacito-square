# SPEC-FR-M4.8: Community Echo Endpoint

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M4.8                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M4                                          |
| Component     | keeper, agent                               |
| Depends On    | SPEC-FR-M8.9, SPEC-FR-M4.7                 |
| Supersedes    | none                                        |

## Context

This spec introduces the first synchronous Keeper-to-Agent messaging capability: a dedicated HTTP endpoint on the Keeper that fans out a user-supplied string message to **all running agents** in a community via NATS, collects a simple decorated reply from each agent, and returns the aggregated results to the caller.

This feature serves two primary purposes:
1. **End-to-end connectivity validation**: a minimal, observable, round-trip proof that the Keeper ↔ NATS ↔ Agent pipeline is alive and that per-agent message routing is functional.
2. **Scale-from-zero smoke path**: if a community's agents are scaled to zero (as managed by SPEC-FR-M8.9), publishing the echo message to the community NATS subject triggers the operator's NATS-driven scale-up before the keeper attempts to collect replies. A configurable wake-up wait time accommodates this.

**Message decoration** by the agent is intentionally trivial in this early phase: the agent wraps the sanitized message in a standard envelope that includes the agent's own `name` and a timestamp, e.g.:
```
[agent:<agentName> at <RFC3339-timestamp>] <sanitized-message>
```

**Sanitization** (performed by the Keeper before publishing) strips non-printable characters and enforces a maximum length of 1000 characters.

**Synchronous fanout**: the Keeper uses the NATS request-reply pattern (`nc.RequestWithContext`) per agent — issuing one request per agent concurrently, then collecting all responses before sending the HTTP reply. The endpoint waits for all agents to reply within a configurable timeout (default: 10 seconds per agent). Agents that do not reply within the timeout contribute a timeout error entry in the response.

**Hexagonal constraint**: the application service must not hold a `*nats.Conn` directly. A new `CommunityBroadcaster` outbound port interface wraps the NATS request-reply logic, keeping the application layer free of infrastructure imports (this also corrects the pre-existing hexagonal violation in `lifecycle_service.go`, which is noted but out of scope to fix here).

## Specification

### 1. Input Sanitization

The Keeper MUST sanitize the end-user message before publishing it to NATS:
- Strip all non-printable Unicode characters (categories Cc, Cs).
- Truncate to a maximum of 1000 characters after stripping.
- If the sanitized message is empty after stripping, return HTTP `400 Bad Request` with `{"error": "message must not be empty after sanitization"}`.

### 2. REST Endpoint

The Keeper MUST expose:

```
POST /api/v1/communities/:community_id/echo
```

- **Auth / Tenant**: governed by the existing `TenantResolutionMiddleware` on the `v1` group.
- **Request Content-Type**: `application/json`.
- **Request body**:
  ```json
  { "message": "<user string>" }
  ```
  `message` is required (`binding:"required"`). Bind using `c.ShouldBindJSON`.
- **Response**: see Section 5 (API Contract).

### 3. Community & Agent Resolution

The application service MUST:
1. Load the `Community` from the repository by `communityID` and tenant context; return `404` if not found.
2. List all `Agent` records assigned to that community (`CommunityID == communityID`) from the `AgentRepository`.
3. Filter agents whose `Status` is `running` (i.e., the agent pod is active). Agents in other states (defined, assigned, pending, stopped, error, terminated) MUST be excluded from the fanout.
4. If no running agents exist after filtering:
   a. Check the community's deployed agent CRD statuses. If at least one agent is in `Idle` phase (scaled to zero), set a `woke_community` flag in the response and still proceed — the NATS message publication to the community subject (per SPEC-FR-M8.9) will trigger scale-up. In this case, the endpoint SHOULD wait up to a configurable `wakeUpWaitSeconds` (default: 30s, configurable via Viper key `keeper.echo.wakeup_wait_seconds`) before attempting the per-agent NATS request.
   b. If no agents are running and none are idle/scalable, return `503 Service Unavailable` with `{"error": "no running agents in community"}`.

### 4. NATS Fanout (CommunityBroadcaster Outbound Port)

A new outbound port MUST be defined:

```go
// EchoRequest is the payload sent to each agent via NATS.
type EchoRequest struct {
    Message     string `json:"message"`
    CommunityID string `json:"community_id"`
    TenantID    string `json:"tenant_id"`
}

// EchoReply is the decorated reply expected from each agent.
type EchoReply struct {
    AgentName   string `json:"agent_name"`
    Decorated   string `json:"decorated"`
    Timestamp   string `json:"timestamp"` // RFC3339
}

// CommunityBroadcaster is the driven outbound port for community-scoped NATS request-reply.
type CommunityBroadcaster interface {
    // RequestEcho sends an EchoRequest to a single agent subject and waits for an EchoReply.
    // Subject format: ts.community.{communityID}.agent.{agentName}
    // Returns an error if the agent does not reply within the deadline.
    RequestEcho(ctx context.Context, communityID, agentName string, req EchoRequest) (*EchoReply, error)
}
```

The `CommunityBroadcaster` MUST be implemented as a NATS adapter at `internal/keeper/adapters/outbound/nats/community_broadcaster.go`. It uses `nc.RequestMsgWithContext` with the subject `ts.community.{communityID}.agent.{agentName}`.

The application service MUST fan out requests to all filtered running agents **concurrently** using goroutines + channels, bounded by `context.Context` with a configurable per-agent timeout (default: 10 seconds, Viper key `keeper.echo.agent_timeout_seconds`). Goroutine results (success or error) are collected into the `AgentEchoResult` list.

### 5. Agent-Side Echo Subscriber

Each agent MUST subscribe to `ts.community.{communityID}.agent.{agentName}` for incoming `EchoRequest` messages.

On receiving an `EchoRequest`:
1. **Log** the sanitized `message` at `info` level, including `agent_name`, `community_id`, `tenant_id`.
2. **Decorate** the message: `[agent:<agentName> at <RFC3339-now>] <message>`.
3. **Reply** to the NATS reply subject with a JSON-encoded `EchoReply`.

The agent's NATS subscriber for echo messages MUST be implemented as an **inbound adapter** at `internal/agent/adapters/inbound/nats/echo_subscriber.go`, following the hexagonal structure of the agent component.

> **Note**: The agent-side implementation spans the `agent` component which is currently in DRAFT (SPEC-FR-M5.x series). For this spec, the agent-side subscriber is **specified here** as a dependency but implementation may be co-delivered with M5 agent scaffolding. The Keeper-side implementation can be verified with a mock subscriber for testing purposes.

### 6. Scale-from-Zero Integration (SPEC-FR-M8.9 Dependency)

The Keeper MUST use the community NATS subject `ts.community.{communityID}.agent.*` pattern implicitly, since per-agent subjects are sub-patterns of this wildcard. The operator's `NATSCommunitySubscriber` (SPEC-FR-M8.9) subscribes to this wildcard and will detect the per-agent echo requests, triggering scale-up of all scaled-to-zero agents in the community.

The Keeper echo endpoint MUST NOT directly interact with the operator or Kubernetes — scale-up is a side-effect of the NATS fanout itself.

### 7. Timeout & Error Handling

- If a per-agent request times out, the response entry for that agent MUST include `"error": "timeout"` (no HTTP error — the endpoint still returns `200 OK` with partial results).
- If the endpoint-level context is cancelled (client disconnect), all in-flight goroutines MUST be aborted via `context.Context` cancellation.
- If `nc` is nil (NATS not configured), return `503 Service Unavailable` with `{"error": "NATS messaging is not available"}`.

### 8. Observability

- OTel span: `keeper.echo_community`, kind `SpanKindServer`, attributes: `community_id`, `tenant_id`, `agent_count`, `timeout_count`.
- Log at `info` on success: `community_id`, `tenant_id`, `agent_count`, `replied_count`, `timeout_count`.
- Log at `warn` for per-agent timeouts: `agent_name`, `community_id`.

## Acceptance Criteria

1. **Happy path**: `POST /api/v1/communities/{id}/echo` with a valid message and at least one running agent returns `200 OK` with one result per running agent, each containing the decorated message and no error.
2. **Sanitization**: A message containing non-printable characters is stripped before publishing; the decorated reply contains the stripped version.
3. **Empty message rejection**: A message that is empty or becomes empty after sanitization returns `400 Bad Request`.
4. **Community not found**: `communityID` that does not exist returns `404 Not Found`.
5. **No running agents**: Community exists but no agents are in `running` state and none are in `Idle` CRD phase returns `503 Service Unavailable`.
6. **Agent timeout**: An agent that does not reply within the per-agent timeout contributes `"error": "timeout"` in its result slot; the response HTTP status is still `200 OK`.
7. **Concurrent fanout**: For a community with 3 running agents, all 3 requests are issued concurrently (total latency is approximately one agent round-trip, not three).
8. **NATS unavailable**: When `nc` is nil, returns `503 Service Unavailable`.
9. **Tenant isolation**: Requests without a valid tenant context return `401 Unauthorized`.
10. **Scale-from-zero side-effect**: When agents are in `Idle` state, the echo NATS messages trigger the operator's scale-up (via SPEC-FR-M8.9); the endpoint waits up to `wakeUpWaitSeconds` before proceeding.
11. **`CommunityBroadcaster` port compliance**: The application service `EchoService` MUST NOT import `nats.go` directly — only the `CommunityBroadcaster` interface.

## Test Plan

### Automated Tests

1. **Unit Tests** — `internal/keeper/application/service/echo_service_test.go` [NEW]:
   - Mock `CommunityBroadcaster`, `AgentRepository`, `CommunityRepository`.
   - Assert fanout is concurrent (use `sync.WaitGroup` assertions in mock).
   - Assert agent with `status != running` is excluded from fanout.
   - Assert per-timeout contributes `"error":"timeout"` in result; HTTP still 200.
   - Assert empty sanitized message returns domain error.
   - Assert `nc == nil` path returns broadcaster-unavailable error.

2. **Unit Tests** — `internal/keeper/domain/model/echo_test.go` [NEW]:
   - `SanitizeMessage` strips non-printable characters.
   - `SanitizeMessage` truncates at 1000 characters.
   - `DecorateMessage(agentName, message)` returns `[agent:<name> at <timestamp>] <message>`.

3. **HTTP Handler Tests** — `internal/keeper/adapters/inbound/http/echo_handlers_test.go` [NEW]:
   - Mock `EchoUseCase`.
   - `POST .../echo` with valid body → `200 OK` with expected JSON structure.
   - `POST .../echo` with empty message → `400 Bad Request`.
   - `POST .../echo` community not found → `404 Not Found`.
   - `POST .../echo` no running agents → `503 Service Unavailable`.
   - `POST .../echo` missing tenant → `401 Unauthorized`.
   - Tests run in `gin.TestMode` through `ServeHTTP`.

4. **Integration Tests** — `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go` [NEW]:
   - Use in-process NATS server.
   - Subscribe a mock reply handler on the agent subject before calling `RequestEcho`.
   - Assert reply is correctly decoded into `EchoReply`.
   - Assert timeout error when no subscriber replies within deadline.

### Manual Verification

1. Deploy keeper + operator to minikube with a community of 2 agents (scaled down via M4.4).
2. `curl -X POST http://keeper:8081/api/v1/communities/{id}/echo -d '{"message":"hello world"}' -H "X-Tenant-ID: tenant1"`.
3. Observe operator logs: both agents should wake up.
4. Observe agent logs: sanitized `hello world` logged at `info`.
5. Confirm response contains two decorated entries with correct format `[agent:<name> at <ts>] hello world`.

## API Contract

### Endpoint

```
POST /api/v1/communities/:community_id/echo
```

### Request

- **Headers**: `Content-Type: application/json`, `X-Tenant-ID: <tenant>`
- **URL Parameter**: `community_id` — UUID of the target community
- **Body**:
  ```json
  { "message": "hello world" }
  ```

### Response — `200 OK`

```json
{
  "community_id": "uuid",
  "woke_community": false,
  "results": [
    {
      "agent_name": "agent-alpha",
      "decorated": "[agent:agent-alpha at 2026-05-29T18:00:00Z] hello world",
      "error": ""
    },
    {
      "agent_name": "agent-beta",
      "decorated": "",
      "error": "timeout"
    }
  ]
}
```

### Response — `400 Bad Request`
```json
{ "error": "message must not be empty after sanitization" }
```

### Response — `404 Not Found`
```json
{ "error": "community not found" }
```

### Response — `503 Service Unavailable`
```json
{ "error": "no running agents in community" }
```
or
```json
{ "error": "NATS messaging is not available" }
```

### OpenAPI Tag

`community/echo` — registered in the keeper's `openapi.json` and `GET /openapi.json`.

## Files Affected

### Domain
- `[NEW] internal/keeper/domain/model/echo.go` — `SanitizeMessage`, `DecorateMessage`, `EchoRequest`, `EchoReply`, `AgentEchoResult`, `CommunityEchoResponse` value types.
- `[NEW] internal/keeper/domain/model/echo_test.go` — Unit tests for sanitization and decoration.

### Application Ports
- `[MODIFY] internal/keeper/application/ports/inbound/usecases.go` — Add `EchoUseCase` interface with `EchoCommunity(ctx, communityID, message) (*CommunityEchoResponse, error)`.
- `[NEW] internal/keeper/application/ports/outbound/community_broadcaster.go` — `CommunityBroadcaster` interface.

### Application Service
- `[NEW] internal/keeper/application/service/echo_service.go` — `EchoServiceImpl` implementing `EchoUseCase`.
- `[NEW] internal/keeper/application/service/echo_service_test.go` — Unit tests with mocked ports.

### Adapters — Outbound
- `[NEW] internal/keeper/adapters/outbound/nats/community_broadcaster.go` — NATS request-reply implementation of `CommunityBroadcaster`.
- `[NEW] internal/keeper/adapters/outbound/nats/community_broadcaster_test.go` — Integration tests with in-process NATS server.

### Adapters — Inbound (Keeper)
- `[NEW] internal/keeper/adapters/inbound/http/echo_handlers.go` — `EchoHandler` with `EchoCommunity` Gin handler. Registers `POST /api/v1/communities/:community_id/echo`.
- `[NEW] internal/keeper/adapters/inbound/http/echo_handlers_test.go` — Handler unit tests in `gin.TestMode`.

### Adapters — Inbound (Agent)
- `[NEW] internal/agent/adapters/inbound/nats/echo_subscriber.go` — NATS subscriber for per-agent echo requests. Logs and replies with decorated message.

### Bootstrap
- `[MODIFY] internal/keeper/bootstrap.go` — Construct `NATSCommunityBroadcaster`, `EchoServiceImpl`, `EchoHandler`; register `POST .../echo` route on the `v1` group.
