# SPEC-FR-M6.0: Event-Driven Architecture Foundation & Conversational Schema

| Field         | Value                                                         |
|---------------|---------------------------------------------------------------|
| ID            | SPEC-FR-M6.0                                                  |
| Status        | VERIFIED                                                      |
| Milestone     | M6                                                            |
| Component     | shared, keeper, agent                                         |
| Depends On    | SPEC-FR-M4.7, SPEC-FR-M5.1, SPEC-FR-M5.3, SPEC-FR-M5.4      |
| Supersedes    | SPEC-FR-M4.8                                                  |

## Context

Milestone 5 delivered the agent cognitive core: LLM reasoning, STM, LTM, and MCP tool invocation.
The only integration path between keeper and agent was the synchronous echo endpoint (SPEC-FR-M4.8),
which relied on a NATS request-reply pattern — a blocking, tightly coupled transport that contradicts
the system's event-driven and reactive principles.

This specification introduces the foundational event-driven architecture that will govern all future
keeper-to-agent and agent-to-client communication. It encompasses three tightly related concerns:

1. **Generic Domain Event Model** — a shared Go package (`pkg/events`) that defines a versioned,
   schema-annotated event envelope with standardized NATS message headers (tenant, schema URI, source,
   instance, trace context). All events in the system — current and future — MUST conform to this model.

2. **Conversational SchemaRef** — a concrete schema identified by URNs in the
   `urn:tacito:schema:conversational:*:v1` namespace, defining three operations that represent a
   complete agent conversation lifecycle: `start-thread`, `add-user-message`, and `end-thread`.
   This schema replaces the echo endpoint as the primary keeper-to-agent interaction channel.

3. **Keeper Event Infrastructure** — two new REST endpoints on the keeper:
   - A **generic fire-and-forget event endpoint** (`POST /api/v1/events`) that accepts any valid
     domain event and publishes it to NATS, returning `202 Accepted` immediately.
   - A **Server-Sent Events (SSE) endpoint** (`GET /api/v1/events/stream`) that subscribes to all
     NATS event subjects and fans events out to connected HTTP clients in real time, enabling
     lightweight event streaming without WebSocket overhead.

4. **Agent Event-Based Brain Dispatch** — the agent's NATS subscriber is redesigned from a named
   echo subscriber to a generic event subscriber that reads the `schemaRef` NATS header to determine
   which brain flow to activate. The conversational schemaRef maps to the existing LLM reasoning
   pipeline. Future schemaRefs (e.g., tools, routing) will add new flows without modifying the
   subscriber itself.

The echo capability (`SPEC-FR-M4.8`) is fully superseded and MUST be removed as part of this spec's
implementation.

---

## Specification

### 1. Shared Event Package (`pkg/events`)

#### 1.1 Generic Event Envelope

The system MUST define a canonical Go struct `DomainEvent` in `pkg/events/event.go`:

```go
// DomainEvent is the canonical, schema-agnostic event envelope.
// Every event published or consumed in the system MUST conform to this structure.
type DomainEvent struct {
    // EventID is a globally unique identifier for this event instance (UUID v4).
    EventID string `json:"event_id"`

    // SchemaRef is the versioned URN identifying the event's payload schema.
    // Format: urn:tacito:schema:{domain}:{operation}:{version}
    // Example: urn:tacito:schema:conversational:start-thread:v1
    SchemaRef string `json:"schema_ref"`

    // Source identifies the component that emitted this event.
    // Format: {componentName}/{instanceID}
    // For agent: agent/{agentID}  (e.g., "agent/f47ac10b-58cc-4372-a567-0e02b2c3d479")
    // For others: {component}/{HOSTNAME}  (e.g., "keeper/pod-abc123")
    Source string `json:"source"`

    // TenantID is the resolved tenant identifier for this event.
    TenantID string `json:"tenant_id"`

    // OccurredAt is the UTC timestamp when the event was created (RFC3339Nano).
    OccurredAt string `json:"occurred_at"`

    // Payload contains the schema-specific event data. Its structure is
    // validated against the schema identified by SchemaRef.
    Payload json.RawMessage `json:"payload"`
}
```

#### 1.2 NATS Header Projection

When a `DomainEvent` is published to NATS, its metadata MUST be projected into NATS message headers
so subscribers can route/filter events without deserializing the full JSON body:

| NATS Header           | Source Field          | Purpose                                      |
|-----------------------|-----------------------|----------------------------------------------|
| `X-Tacito-Schema`     | `SchemaRef`           | Event type routing                           |
| `X-Tacito-Source`     | `Source`              | Originating component/instance               |
| `X-Tacito-Tenant`     | `TenantID`            | Tenant isolation                             |
| `X-Tacito-Event-ID`   | `EventID`             | Deduplication / idempotency key              |
| `X-Tacito-Occurred`   | `OccurredAt`          | Ordering / observability                     |
| `Traceparent`         | OTel span context     | W3C distributed tracing propagation          |

#### 1.3 NATS Subject Convention

Events MUST be published to the following NATS subject structure:

```
ts.community.{communityID}.agent.{agentName}
```

This preserves the existing per-agent community subject namespacing (SPEC-FR-M6.3). The `SchemaRef`
NATS header (not the subject) determines the event type and the brain flow to activate.

#### 1.4 Source Construction

The event `Source` field MUST be constructed by the emitting component as:
```
{componentName}/{instanceID}
```
where:
- `{componentName}` is one of: `keeper`, `bff`, `operator`.
- For the **agent component**, `{componentName}` is `agent` and `{instanceID}` MUST be the agent's
  logical ID (UUID, read from the `AGENT_ID` environment variable set by the operator on the pod).
  This is required because multiple distinct agents may run within the same Kubernetes replica set
  or community, and the pod hostname would not uniquely identify the agent.
- For all other components (`keeper`, `bff`, `operator`), `{instanceID}` is read from the
  `HOSTNAME` environment variable (set by Kubernetes to the pod name). In local development,
  it SHOULD fall back to the machine hostname.

**Examples**:
- `keeper/pod-abc123` — a keeper pod
- `agent/f47ac10b-58cc-4372-a567-0e02b2c3d479` — agent identified by its logical UUID
- `bff/pod-xyz789` — a BFF pod

#### 1.5 Schema Registry Constants

The shared package MUST define Go string constants for all known SchemaRefs to prevent magic strings:

```go
const (
    SchemaConversationalStartThread   = "urn:tacito:schema:conversational:start-thread:v1"
    SchemaConversationalAddUserMessage = "urn:tacito:schema:conversational:add-user-message:v1"
    SchemaConversationalEndThread      = "urn:tacito:schema:conversational:end-thread:v1"
)
```

#### 1.6 Event Builder Helpers

The package MUST expose a constructor `NewDomainEvent(schemaRef, source, tenantID string, payload any) (DomainEvent, error)`
that:
- Generates a new UUID v4 as `EventID`.
- Sets `OccurredAt` to `time.Now().UTC()` in RFC3339Nano format.
- JSON-marshals `payload` into `Payload`.
- Validates that `schemaRef`, `source`, and `tenantID` are non-empty.

> **Return by value**: `DomainEvent` is immutable after construction. Returning a value type (not
> a pointer) avoids heap allocation for the common case and prevents accidental mutation by callers.
> Callers that need a pointer for interface satisfaction may take the address of the returned value.

This **auto-population policy** — system-generated `EventID`, `OccurredAt`, and `Source` — applies
universally: it MUST be enforced both at the keeper REST ingress (Section 3.1) and whenever any
component (agent, keeper, bff) creates and publishes new events internally (e.g., an agent emitting
an `agent-response` event).

---

### 2. Conversational Schema Payloads (`pkg/events`)

The following Go structs MUST be defined in `pkg/events/conversational.go` as the typed payloads
for the three conversational operations. They are the `payload` field of a `DomainEvent`.

#### 2.1 `start-thread` Payload

```go
// StartThreadPayload is the payload for urn:tacito:schema:conversational:start-thread:v1.
type StartThreadPayload struct {
    // ThreadID is a client-supplied or keeper-generated UUID for this conversation thread.
    ThreadID string `json:"thread_id"`

    // CommunityID is the UUID of the target community.
    CommunityID string `json:"community_id"`

    // Metadata is an optional arbitrary key-value map for caller-defined context
    // (e.g., user ID, session ID). Values MUST be strings.
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

#### 2.2 `add-user-message` Payload

```go
// AddUserMessagePayload is the payload for urn:tacito:schema:conversational:add-user-message:v1.
type AddUserMessagePayload struct {
    // ThreadID identifies the active conversation thread. MUST match a previously started thread.
    ThreadID string `json:"thread_id"`

    // CommunityID is the UUID of the target community.
    CommunityID string `json:"community_id"`

    // Message is the user's input text. MUST be non-empty after sanitization.
    // Sanitization (strip non-printable Unicode, max 4096 characters) is performed by the keeper
    // before building the event; the agent receives an already-sanitized message.
    Message string `json:"message"`
}
```

#### 2.3 `end-thread` Payload

```go
// EndThreadPayload is the payload for urn:tacito:schema:conversational:end-thread:v1.
type EndThreadPayload struct {
    // ThreadID identifies the conversation thread to close.
    ThreadID string `json:"thread_id"`

    // CommunityID is the UUID of the target community.
    CommunityID string `json:"community_id"`

    // Reason is an optional human-readable string describing why the thread ended.
    Reason string `json:"reason,omitempty"`
}
```

---

### 3. Keeper: Generic Fire-and-Forget Event Endpoint

#### 3.1 REST Endpoint

The Keeper MUST expose:

```
POST /api/v1/events
```

- **Auth / Tenant**: governed by the existing `TenantResolutionMiddleware` on the `v1` group.
- **Request Content-Type**: `application/json`.
- **Request body**: a partial event containing ONLY `schema_ref` and `payload`. The fields
  `tenant_id`, `source`, `event_id`, and `occurred_at` MUST be **omitted** from the request body;
  they are always system-generated and MUST NOT be accepted from the caller. An unrecognised field
  (e.g., `event_id` in the request) MUST be silently ignored (not rejected).
- **System auto-population** (enforced by the keeper before NATS publication):
  - `tenant_id` — set from the tenant resolved by `TenantResolutionMiddleware`.
  - `source` — set to `keeper/{instanceID}` (Section 1.4).
  - `event_id` — always a new UUID v4 generated by the keeper.
  - `occurred_at` — always `time.Now().UTC()` in RFC3339Nano format.
- **Validation**:
  - `schema_ref` MUST be non-empty and match the URN format `urn:tacito:schema:*`.
  - `payload` MUST be a valid JSON object or array (not null).
  - For the conversational schema family, the keeper MUST validate the payload against the typed
    struct (binding); invalid payloads return `400 Bad Request`.
- **NATS Routing**: the keeper determines the target NATS subject from the `schema_ref` and event
  payload content:
  - For `urn:tacito:schema:conversational:*`: extract `community_id` and `agent_name` from the
    payload; subject = `ts.community.{communityID}.agent.{agentName}`.
  - For unknown schemaRefs: publish to a generic topic `ts.events.{tenantID}` for observability.
- **Response**: `202 Accepted` with the fully populated `DomainEvent` (all system fields filled in).

#### 3.2 Message Sanitization for Conversational Events

When the keeper receives an `add-user-message` event:
- Strip all non-printable Unicode characters (categories Cc, Cs) from `Message`.
- Truncate to a maximum of 4096 characters after stripping.
- If the sanitized message is empty, return `400 Bad Request` with
  `{"error": "message must not be empty after sanitization"}`.
- The sanitized message replaces the original in the payload before NATS publication.

#### 3.3 Outbound Port

An outbound port MUST be defined in `internal/keeper/application/ports/outbound/event_publisher.go`:

```go
// EventPublisher is the driven outbound port for publishing domain events to the message bus.
type EventPublisher interface {
    // Publish publishes a DomainEvent to the message bus for the given target subject.
    // Returns an error only if the publish operation itself fails (e.g., NATS connection lost).
    Publish(ctx context.Context, subject string, event events.DomainEvent) error
}
```

The NATS adapter implementing `EventPublisher` MUST:
- Serialize the `DomainEvent` to JSON as the message body.
- Project all header fields (Section 1.2) into NATS message headers.
- Inject OTel W3C `traceparent` header from the active span in `ctx`.

---

### 4. Keeper: Server-Sent Events (SSE) Stream Endpoint

#### 4.1 REST Endpoint

The Keeper MUST expose:

```
GET /api/v1/events/stream
```

- **Auth / Tenant**: governed by `TenantResolutionMiddleware`.
- **Response Content-Type**: `text/event-stream` with headers `Cache-Control: no-cache`,
  `Connection: keep-alive`, `X-Accel-Buffering: no`.
- **Subscription**: on connection, the keeper subscribes to the NATS hierarchical wildcard subject
  `ts.community.>` filtered by `X-Tacito-Tenant` header matching the resolved tenantID.
  This wildcard captures both inbound events (`ts.community.*.agent.*`) and agent response events
  (`ts.community.*.agent.*.response`), ensuring the SSE stream is a complete bidirectional channel.
  Tenant isolation is enforced by checking the `X-Tacito-Tenant` NATS header on each message.
- **Event format**: each NATS message received MUST be forwarded as an SSE event:
  ```
  id: {event_id}
  event: {schemaRef-last-segment}   (e.g., "add-user-message")
  data: {full DomainEvent JSON}
  
  ```
  (SSE standard two-newline terminator after each event block.)
- **Heartbeat**: the keeper MUST send an SSE comment (`: keepalive`) every 15 seconds to prevent
  proxy timeouts. Interval MUST be configurable via Viper key `keeper.sse.heartbeat_seconds`
  (default: 15).
- **Client disconnect**: when the HTTP client disconnects, the NATS subscription MUST be drained and
  the goroutine MUST terminate cleanly via `context.Context` cancellation.
- **Inbound Port**: a new `EventStreamUseCase` inbound port MUST be defined to decouple the Gin
  handler from subscription logic.

#### 4.2 Outbound Subscription Port

An outbound port MUST be defined in `internal/keeper/application/ports/outbound/event_subscriber.go`:

```go
// EventSubscription represents an active NATS subscription that can be stopped.
type EventSubscription interface {
    Stop() error
}

// EventSubscriber is the driven outbound port for subscribing to domain event streams.
type EventSubscriber interface {
    // Subscribe creates a wildcard subscription on the given subject pattern.
    // Events matching tenantID are forwarded to the provided handler.
    // Returns an EventSubscription that the caller MUST Stop() on cleanup.
    Subscribe(ctx context.Context, subjectPattern string, tenantID string, handler func(*events.DomainEvent)) (EventSubscription, error)
}
```

---

### 5. Agent: Generic Event Subscriber

#### 5.1 Replacement of Echo Subscriber

The `EchoSubscriber` (`internal/agent/adapters/inbound/nats/echo_subscriber.go`) MUST be deleted
and replaced by a `EventSubscriber` in `internal/agent/adapters/inbound/nats/event_subscriber.go`.

The new subscriber:
- Subscribes to `ts.community.{communityID}.agent.{agentName}` (same subject as before).
- On message receipt, reads the `X-Tacito-Schema` NATS header to determine the schemaRef.
- Dispatches to the appropriate brain flow via a `SchemaRouter` inbound port (see Section 5.2).

#### 5.2 Schema Router Inbound Port

A new inbound port MUST be defined in `internal/agent/application/ports/inbound/schema_router.go`:

```go
// SchemaRouter dispatches incoming domain events to the appropriate brain flow
// based on the event's schemaRef.
type SchemaRouter interface {
    // Route dispatches the event to the appropriate handler based on SchemaRef.
    // Returns an error if no handler is registered for the given SchemaRef,
    // or if the handler itself fails.
    Route(ctx context.Context, event *events.DomainEvent) error
}
```

The `SchemaRouter` implementation MUST be registered in the agent bootstrap and accept handlers
for each known schemaRef. Unknown schemaRefs MUST be logged at `warn` level and dropped (no error
returned to NATS — fire-and-forget semantics).

#### 5.3 Conversational Flow Handlers

The agent MUST register the following handlers in the `SchemaRouter`:

| SchemaRef                                              | Handler Behavior                                                                                                  |
|--------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| `urn:tacito:schema:conversational:start-thread:v1`     | Extract `ThreadID` from payload; initialize STM conversation state in Redis for `{threadID}`. Log at `info`.     |
| `urn:tacito:schema:conversational:add-user-message:v1` | Extract `ThreadID` + `Message`; invoke `MessageProcessor.ProcessIncomingMessage(ctx, tenantID, agentName, threadID, message)`. Log at `info`. On LLM failure, MUST roll back any STM entry written for that turn (see Section 5.5). The LLM result MUST be published back as a `urn:tacito:schema:conversational:agent-response:v1` event (see Section 5.6). |
| `urn:tacito:schema:conversational:end-thread:v1`       | Extract `ThreadID`; flush STM (delete Redis keys for `threadID`); if LTM is enabled, persist thread summary to Qdrant. Log at `info`. |

> **Note on response flow**: The full agent-to-keeper streaming pipeline is deferred to a follow-up
> M6.x spec. However, M6.0 **reserves** the `agent-response` schemaRef and defines its payload
> contract (Section 5.6) so the SSE stream can carry it once the agent side is implemented.

#### 5.4 Large Message Offloading

The S3 blob offloading logic (currently embedded in `echo_subscriber.go`) MUST be preserved and
moved to a shared utility in `internal/agent/adapters/inbound/nats/offload.go`. The `add-user-message`
handler in the `SchemaRouter` MUST invoke this utility when `len(message) > 256*1024` and
`BlobStore` is configured.

#### 5.5 STM Rollback on LLM Failure

When the `add-user-message` handler invokes `MessageProcessor.ProcessIncomingMessage` and the LLM
pipeline returns an error:

1. The agent MUST identify the STM entry (Redis key) written for the current user turn during this
   invocation.
2. The agent MUST delete that STM entry (rollback the turn) before propagating the error upward.
3. The rollback MUST be best-effort: if the Redis delete itself fails, the agent MUST log an `error`
   with both the original LLM error and the rollback failure, but MUST NOT panic or block.
4. After rollback, the agent MUST log a `warn` with `event_id`, `thread_id`, `agent_name`, and
   the LLM error message.

This ensures a failed turn does not corrupt the conversation history in STM, enabling safe retry
by the caller (re-sending the `add-user-message` event).

#### 5.6 `agent-response` SchemaRef — Active Publication

The following schemaRef and payload MUST be defined as constants and structs in `pkg/events/conversational.go`
and MUST be **actively produced** by the agent after every successful `add-user-message` processing.

```go
const SchemaConversationalAgentResponse = "urn:tacito:schema:conversational:agent-response:v1"

// AgentResponsePayload is the payload for urn:tacito:schema:conversational:agent-response:v1.
// Published by the agent after completing LLM reasoning for an add-user-message event.
type AgentResponsePayload struct {
    // ThreadID correlates this response to its originating thread.
    ThreadID string `json:"thread_id"`

    // CommunityID is the UUID of the agent's community.
    CommunityID string `json:"community_id"`

    // AgentName is the name of the responding agent.
    AgentName string `json:"agent_name"`

    // CorrelationEventID is the EventID of the add-user-message event that triggered this response.
    CorrelationEventID string `json:"correlation_event_id"`

    // Response is the agent's LLM-generated reply text.
    Response string `json:"response"`

    // Finished indicates whether this is the final (complete) response chunk.
    // Set to true for single-shot responses; reserved for streaming chunked responses in future specs.
    Finished bool `json:"finished"`
}
```

**Agent publication behaviour**: after `MessageProcessor.ProcessIncomingMessage` returns successfully,
the agent MUST:
1. Construct an `AgentResponsePayload` with the LLM result, correlating it to the triggering event
   via `CorrelationEventID`.
2. Call `NewDomainEvent(SchemaConversationalAgentResponse, source, tenantID, payload)` — where
   `source` is `agent/{agentID}` (Section 1.4) and all other system fields are auto-populated.
3. Publish the event to a **keeper-facing response subject**:
   `ts.community.{communityID}.agent.{agentName}.response`
   The keeper's SSE wildcard subscription (`ts.community.*.agent.*`) MUST be broadened to
   `ts.community.>` to also capture this sub-level response subject.
4. Set `Finished: true` for single-shot responses (current behaviour). Streaming chunks are
   reserved for a future spec.

**Auto-population policy for agent-emitted events**: the same system fields (`event_id`,
`occurred_at`, `source`) MUST be generated fresh by the agent when constructing the response event;
no fields from the triggering inbound event are copied except `CorrelationEventID`.

The keeper's SSE endpoint forwards `agent-response` events to connected SSE clients alongside
publisher events — the SSE stream is therefore the complete bidirectional event channel.



---

### 6. Removal of Echo Infrastructure

The following artifacts from SPEC-FR-M4.8 MUST be deleted:

- `internal/agent/adapters/inbound/nats/echo_subscriber.go`
- `internal/agent/adapters/inbound/nats/echo_subscriber_test.go`
- `internal/keeper/domain/model/echo.go` (SanitizeMessage, DecorateMessage, EchoRequest, EchoReply)
- `internal/keeper/domain/model/echo_test.go`
- `internal/keeper/application/ports/outbound/community_broadcaster.go`
- `internal/keeper/application/service/echo_service.go`
- `internal/keeper/application/service/echo_service_test.go`
- `internal/keeper/adapters/outbound/nats/community_broadcaster.go`
- `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go`
- `internal/keeper/adapters/inbound/http/echo_handlers.go`
- `internal/keeper/adapters/inbound/http/echo_handlers_test.go`

The POST `/api/v1/communities/:community_id/echo` route registration in keeper `bootstrap.go` and
all related wiring MUST be removed.

---

### 7. Observability

#### 7.1 Keeper

- OTel span `keeper.publish_event` on the fire-and-forget endpoint; attributes: `schema_ref`,
  `tenant_id`, `community_id` (if conversational), `agent_name` (if routed per-agent).
- Log at `info` on successful publish: `event_id`, `schema_ref`, `tenant_id`, `subject`.
- Log at `warn` on NATS publish failure.
- Prometheus counter `keeper_events_published_total` labelled by `schema_ref` and `status`
  (`success`/`error`).

#### 7.2 Keeper SSE

- Log at `info` when a new SSE client connects and when it disconnects.
- Prometheus gauge `keeper_sse_active_connections` (no labels).
- Prometheus counter `keeper_sse_events_forwarded_total` labelled by `schema_ref`.

#### 7.3 Agent

- Reuse existing Prometheus counters `agent_nats_messages_processed_total` and
  `agent_nats_processing_duration_seconds` (already defined in observability package).
- Log at `info` for each handled conversational event; include `event_id`, `schema_ref`,
  `thread_id`, `tenant_id`, `agent_name`.
- Log at `warn` for unknown schemaRef with `schema_ref` field; the event is dropped.

---

## Acceptance Criteria

1. **Event model completeness**: `pkg/events.DomainEvent` compiles with all required fields; `NewDomainEvent` returns a valid envelope with UUID `EventID`, RFC3339Nano `OccurredAt`, and marshalled `Payload`.
2. **Value return**: `NewDomainEvent` returns `(DomainEvent, error)` — not a pointer; callers may take the address if needed.
3. **Header projection**: publishing a `DomainEvent` to NATS results in a message with all six mandatory NATS headers set correctly.
4. **SchemaRef constants**: all four conversational schema URNs (including `agent-response`) are defined as constants and used throughout the codebase (no magic strings).
5. **Fire-and-forget endpoint — request contract**: `POST /api/v1/events` accepts only `schema_ref` and `payload` in the request body; any presence of `tenant_id`, `source`, `event_id`, or `occurred_at` in the body is silently ignored.
6. **Fire-and-forget endpoint — response**: a valid `add-user-message` event returns `202 Accepted` with all four system fields auto-populated; NATS receives one message on `ts.community.{id}.agent.{name}`.
7. **Sanitization**: an `add-user-message` event with a message containing non-printable characters is sanitized before publish; a message that is empty after sanitization returns `400 Bad Request`.
8. **Source auto-set**: the `source` in the `202` response is always `keeper/{instanceID}`, never whatever the caller may have sent.
9. **Agent source**: agent-emitted events carry `agent/{agentID}` as source, where `agentID` matches the `AGENT_ID` env var, not the pod hostname.
10. **SSE streaming**: `GET /api/v1/events/stream` keeps the connection open and emits events as the keeper publishes them; a heartbeat comment appears every `heartbeat_seconds`.
11. **SSE wildcard**: the SSE subscription uses `ts.community.>` — both inbound events and agent response events appear on the stream.
12. **SSE tenant filtering**: SSE events belonging to a different tenant are NOT forwarded to the connected client.
13. **SSE cleanup**: disconnecting the SSE client stops the underlying NATS subscription within one heartbeat interval.
14. **Agent start-thread**: receiving a `start-thread` event initializes STM state in Redis; subsequent `add-user-message` events on the same `threadID` access the correct history.
15. **Agent add-user-message**: the LLM pipeline is invoked with the correct `threadID` and sanitized message; the agent does not reply to NATS directly (no reply subject expected from the inbound event).
16. **Agent agent-response publication**: after a successful `add-user-message`, the agent publishes an `agent-response` event to `ts.community.{id}.agent.{name}.response`; this event appears on the keeper SSE stream.
17. **Agent agent-response auto-population**: the agent-emitted `agent-response` event has fresh `event_id` and `occurred_at`; only `correlation_event_id` is copied from the triggering event.
18. **Agent end-thread**: receiving `end-thread` removes STM Redis keys for `threadID`; if LTM is enabled, a summary entry is written to Qdrant.
19. **STM rollback**: when LLM processing fails during `add-user-message`, the STM entry written for that turn is deleted before the error propagates.
20. **Unknown schemaRef**: the agent logs a `warn` and drops events with unregistered schemaRefs; no error is returned to NATS.
21. **Echo removal**: the route `POST /api/v1/communities/:community_id/echo` returns `404 Not Found` (route unregistered); all echo-related source files are absent from the repository.
22. **Hexagonal compliance**: the keeper application service depends only on `EventPublisher` and `EventSubscriber` interfaces; no `*nats.Conn` leaks into the application layer.
23. **OpenAPI**: the keeper's `GET /openapi.json` documents both new endpoints with tag `events/publish` and `events/stream`; the echo endpoint is absent.

---

## Test Plan

### Automated Tests

1. **Unit Tests — `pkg/events/event_test.go`** [NEW]:
   - `NewDomainEvent` returns a valid envelope with UUID, timestamp, marshalled payload.
   - `NewDomainEvent` returns an error if `schemaRef`, `source`, or `tenantID` is empty.
   - `NewDomainEvent` returns an error if `payload` cannot be marshalled.

2. **Unit Tests — `pkg/events/conversational_test.go`** [NEW]:
   - Verify each payload struct JSON round-trips correctly.
   - Verify `StartThreadPayload.Metadata` is omitted when empty.

3. **Unit Tests — `internal/keeper/application/service/event_service_test.go`** [NEW]:
   - Mock `EventPublisher`; assert `Publish` is called with correct subject and headers for each schemaRef.
   - Assert sanitization strips non-printable characters from `add-user-message`.
   - Assert empty-after-sanitize returns domain error.
   - Assert `TenantID` mismatch returns authorization error.
   - Assert `Source` is always overwritten by the keeper.

4. **HTTP Handler Tests — `internal/keeper/adapters/inbound/http/event_handlers_test.go`** [NEW]:
   - `POST /api/v1/events` with valid body → `202 Accepted` with populated envelope.
   - `POST /api/v1/events` with empty message after sanitization → `400 Bad Request`.
   - `POST /api/v1/events` with unknown schemaRef → `202 Accepted` (published to generic topic).
   - `POST /api/v1/events` with mismatched `TenantID` → `403 Forbidden`.
   - `GET /api/v1/events/stream` → `200 OK`, `Content-Type: text/event-stream`.
   - Tests run in `gin.TestMode` through `ServeHTTP`.

5. **Integration Tests — NATS adapter** [NEW]  
   `internal/keeper/adapters/outbound/nats/event_publisher_test.go`:
   - Use in-process NATS server.
   - Publish a `DomainEvent` and assert all NATS headers are set.
   - Assert tenant filter: subscribe to wildcard, publish two events for different tenants, verify only matching tenant event is received.

6. **Unit Tests — `internal/agent/adapters/inbound/nats/event_subscriber_test.go`** [NEW]:
   - Mock `SchemaRouter`; assert `Route` is called with correct `DomainEvent` for each known schemaRef.
   - Assert unknown schemaRef logs `warn` and does NOT call `Route`.

7. **Unit Tests — `internal/agent/application/service/schema_router_test.go`** [NEW]:
   - Assert `start-thread` handler initializes STM.
   - Assert `add-user-message` handler calls `MessageProcessor.ProcessIncomingMessage`.
   - Assert `add-user-message` handler rolls back STM entry when LLM pipeline returns an error.
   - Assert `add-user-message` STM rollback is best-effort: if Redis delete fails, the handler logs `error` but does not panic.
   - Assert `end-thread` handler flushes STM and (when enabled) persists to LTM.
   - Assert unregistered schemaRef returns no error but drops the event.

8. **Unit Tests — `pkg/events/conversational_test.go`** [NEW — extended]:
   - Verify `AgentResponsePayload` JSON round-trips with all fields including `finished: false`.
   - Verify `SchemaConversationalAgentResponse` constant matches the expected URN string.


### Manual Verification

1. Deploy keeper + one agent to minikube with an existing community.
2. `curl -X POST http://keeper:8080/api/v1/events \
     -H "Content-Type: application/json" \
     -H "X-Tenant-ID: tenant1" \
     -d '{"schema_ref":"urn:tacito:schema:conversational:start-thread:v1","payload":{"thread_id":"t1","community_id":"{id}","metadata":{}}}'`
   → Observe `202 Accepted`; agent logs show `start-thread` received.
3. Open SSE stream: `curl http://keeper:8080/api/v1/events/stream -H "X-Tenant-ID: tenant1"`.
4. Fire an `add-user-message` event; observe SSE client receives the event line within 1 second.
5. Fire an `end-thread` event; observe agent logs show STM flush and (if LTM enabled) Qdrant write.
6. Confirm `POST /api/v1/communities/{id}/echo` returns `404 Not Found`.
7. **OpenAPI check**: `curl http://keeper:8080/openapi.json` — verify both `POST /api/v1/events`
   and `GET /api/v1/events/stream` are present with tags `events/publish` and `events/stream`;
   verify the echo endpoint is absent from the spec.


---

## API Contract

### POST /api/v1/events

**Request**
- Headers: `Content-Type: application/json`, `X-Tenant-ID: <tenant>`
- Body — callers provide **only** `schema_ref` and `payload`; system fields are auto-populated:
  ```json
  {
    "schema_ref": "urn:tacito:schema:conversational:add-user-message:v1",
    "payload": {
      "thread_id": "uuid-thread",
      "community_id": "uuid-community",
      "agent_name": "agent-alpha",
      "message": "Hello, agent!"
    }
  }
  ```
  > `tenant_id`, `source`, `event_id`, and `occurred_at` are **never** part of the request body.

**Response — `202 Accepted`** — fully populated envelope returned to caller:
```json
{
  "event_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "schema_ref": "urn:tacito:schema:conversational:add-user-message:v1",
  "source": "keeper/pod-abc123",
  "tenant_id": "tenant1",
  "occurred_at": "2026-06-06T08:00:00.123456789Z",
  "payload": {
    "thread_id": "uuid-thread",
    "community_id": "uuid-community",
    "agent_name": "agent-alpha",
    "message": "Hello, agent!"
  }
}
```

**Response — `400 Bad Request`**
```json
{ "error": "message must not be empty after sanitization" }
```

**Response — `422 Unprocessable Entity`** (unknown or malformed `schema_ref`)
```json
{ "error": "invalid schema_ref: must match urn:tacito:schema:*" }
```

### GET /api/v1/events/stream

**Request**
- Headers: `X-Tenant-ID: <tenant>` (or resolved from JWT in authenticated deployments)

**Response — `200 OK`**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no

id: uuid-v4
event: add-user-message
data: {"event_id":"uuid-v4","schema_ref":"urn:tacito:schema:conversational:add-user-message:v1","source":"keeper/pod-abc123","tenant_id":"tenant1","occurred_at":"2026-06-06T08:00:00Z","payload":{...}}

: keepalive

```

### OpenAPI Tags

- `events/publish` — fire-and-forget event publication.
- `events/stream` — SSE event stream subscription.

---

## Files Affected

### Shared Package (`pkg/events`) — [NEW]
- `[NEW] pkg/events/event.go` — `DomainEvent`, `NewDomainEvent`, source/header utilities.
- `[NEW] pkg/events/conversational.go` — `StartThreadPayload`, `AddUserMessagePayload`, `EndThreadPayload`, `AgentResponsePayload` + all schema URN constants (including reserved `SchemaConversationalAgentResponse`).
- `[NEW] pkg/events/event_test.go` — Unit tests for event construction and validation.
- `[NEW] pkg/events/conversational_test.go` — JSON round-trip tests for all payload structs including `AgentResponsePayload`.

### Keeper — Application Ports
- `[NEW] internal/keeper/application/ports/outbound/event_publisher.go` — `EventPublisher` interface.
- `[NEW] internal/keeper/application/ports/outbound/event_subscriber.go` — `EventSubscriber`, `EventSubscription` interfaces.
- `[MODIFY] internal/keeper/application/ports/inbound/usecases.go` — Add `EventUseCase` and `EventStreamUseCase` interfaces; remove `EchoUseCase`.
- `[DELETE] internal/keeper/application/ports/outbound/community_broadcaster.go`

### Keeper — Application Service
- `[NEW] internal/keeper/application/service/event_service.go` — `EventServiceImpl` implementing `EventUseCase`.
- `[NEW] internal/keeper/application/service/event_service_test.go` — Unit tests with mocked ports.
- `[DELETE] internal/keeper/application/service/echo_service.go`
- `[DELETE] internal/keeper/application/service/echo_service_test.go`

### Keeper — Domain
- `[DELETE] internal/keeper/domain/model/echo.go`
- `[DELETE] internal/keeper/domain/model/echo_test.go`

### Keeper — Adapters Outbound
- `[NEW] internal/keeper/adapters/outbound/nats/event_publisher.go` — NATS adapter implementing `EventPublisher`.
- `[NEW] internal/keeper/adapters/outbound/nats/event_publisher_test.go` — Integration tests with in-process NATS.
- `[NEW] internal/keeper/adapters/outbound/nats/event_subscriber.go` — NATS wildcard adapter implementing `EventSubscriber`.
- `[DELETE] internal/keeper/adapters/outbound/nats/community_broadcaster.go`
- `[DELETE] internal/keeper/adapters/outbound/nats/community_broadcaster_test.go`

### Keeper — Adapters Inbound
- `[NEW] internal/keeper/adapters/inbound/http/event_handlers.go` — `EventHandler` with `PublishEvent` (POST) and `StreamEvents` (GET SSE) Gin handlers. Registers routes on `v1` group.
- `[NEW] internal/keeper/adapters/inbound/http/event_handlers_test.go` — Handler unit tests in `gin.TestMode`.
- `[DELETE] internal/keeper/adapters/inbound/http/echo_handlers.go`
- `[DELETE] internal/keeper/adapters/inbound/http/echo_handlers_test.go`

### Keeper — Bootstrap
- `[MODIFY] internal/keeper/bootstrap.go` — Remove echo wiring; add `NATSEventPublisher`, `NATSEventSubscriber`, `EventServiceImpl`, `EventHandler`; register `POST /api/v1/events` and `GET /api/v1/events/stream`.

### Agent — Adapters Inbound
- `[NEW] internal/agent/adapters/inbound/nats/event_subscriber.go` — Generic event subscriber replacing `EchoSubscriber`.
- `[NEW] internal/agent/adapters/inbound/nats/event_subscriber_test.go` — Unit tests with mock `SchemaRouter`.
- `[NEW] internal/agent/adapters/inbound/nats/offload.go` — Extracted S3 blob offloading utility.
- `[DELETE] internal/agent/adapters/inbound/nats/echo_subscriber.go`
- `[DELETE] internal/agent/adapters/inbound/nats/echo_subscriber_test.go`

### Agent — Application Ports
- `[NEW] internal/agent/application/ports/inbound/schema_router.go` — `SchemaRouter` interface.

### Agent — Application Service
- `[NEW] internal/agent/application/service/schema_router_impl.go` — `SchemaRouterImpl` with handler registration for all three conversational schemaRefs.
- `[NEW] internal/agent/application/service/schema_router_impl_test.go` — Unit tests for each schemaRef handler.

### Agent — Bootstrap
- `[MODIFY] internal/agent/bootstrap.go` — Remove `EchoSubscriber` re-export; expose `EventSubscriber` and `SchemaRouter` constructors.
- `[MODIFY] cmd/agent/main.go` (or equivalent entrypoint) — Wire `SchemaRouterImpl` with all handlers; replace `EchoSubscriber` with `EventSubscriber`.
