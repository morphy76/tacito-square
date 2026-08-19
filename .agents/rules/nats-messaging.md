---
trigger: glob
globs: ["**/*.go"]
description: Event-driven messaging standards, NATS subject namespacing, and message envelopes.
---

# Event-Driven Messaging Standards (NATS)

This rule establishes patterns and requirements for asynchronous, event-driven communication across Tacito Square using **NATS (`github.com/nats-io/nats.go`)**.

## 1. Subject Namespace Hierarchy

All NATS subjects MUST follow the strict, hierarchical namespacing scheme partitioned by tenant:

```text
tacito.<tenant_id>.<domain_scope>.<entity_id>.<event_action>
```

### Standard Subject Patterns:

- **Community Events (Broadcast & Routing)**:
  - `tacito.<tenant_id>.community.<community_id>.events` — General community broadcast
  - `tacito.<tenant_id>.community.<community_id>.agent.hub` — Messages routed to the Hub coordinator
  - `tacito.<tenant_id>.community.<community_id>.registry.request` — Agent card discovery & registry sync
- **Agent Direct Inbox & Observability**:
  - `tacito.<tenant_id>.agent.<agent_id>.inbox` — Direct point-to-point agent communication
  - `tacito.<tenant_id>.agent.<agent_id>.heartbeat` — Periodic agent health & heartbeat broadcast
  - `tacito.<tenant_id>.agent.<agent_id>.thread.<thread_id>.reasoning` — Real-time reasoning trace stream

> [!CAUTION]
> **No Hardcoded Subject Strings**: Always construct NATS subjects using centralized helper functions or constants in `internal/shared/` or the domain/ports layer (e.g. `subject.ForAgentInbox(tenant, agentID)`). Never format raw string literals ad-hoc across handler files.

## 2. Standard Message Envelope

All domain payloads published over NATS MUST be wrapped in a standardized event envelope struct:

```go
type EventEnvelope struct {
    EventID     string          `json:"event_id"`
    EventType   string          `json:"event_type"`
    TenantID    string          `json:"tenant_id"`
    Timestamp   time.Time       `json:"timestamp"`
    TraceParent string          `json:"traceparent,omitempty"`
    Payload     json.RawMessage `json:"payload"`
}
```

- **Trace Context Propagation**: Outbound NATS publishers MUST inject the active OpenTelemetry `traceparent` header or envelope field.
- **Tenant ID**: The `TenantID` must always be present and validated by inbound subscribers before processing.

## 3. Subscriber Lifecycle & Goroutine Safety

- **Non-blocking Message Handlers**: Inbound message handlers (`nats.MsgHandler`) must never perform long-running blocking operations synchronously in the NATS connection thread. Offload heavy processing to worker goroutines with proper queue limits and context management.
- **Graceful Unsubscription**: All subscriptions (`*nats.Subscription`) must be tracked and cleanly closed during component graceful shutdown.
- **Error Handling**: Log subscriber errors with structured metadata (subject, event ID, tenant ID) via zerolog.

---

## Developer Checklists & Verifications

- [ ] Are NATS subjects generated through centralized subject builders?
- [ ] Does the message envelope include `tenant_id` and `traceparent`?
- [ ] Are subscriptions cleanly drained and closed during component shutdown?
