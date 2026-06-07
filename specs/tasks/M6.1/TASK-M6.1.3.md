# TASK-M6.1.3: Agent Hub NATS Subscriptions & Queue Groups

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.1.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M6.1                                |
| Depends On    | TASK-M6.1.2                                 |

## Description

Configure the agent's NATS subscriber logic. When `TS_AGENT_ROLE` is set to `hub`, the agent must subscribe to the inbound subject using a NATS Queue Group to distribute traffic across replica instances.

## Boundary & Target Functions

- **Package**: `internal/agent/adapters/inbound/nats`
- **Files**:
  - `internal/agent/adapters/inbound/nats/event_subscriber.go`
  - `cmd/agent/main.go`

## Work Items

1. **RED Phase (Write Tests First)**:
   * Write unit/integration tests in `internal/agent/adapters/inbound/nats/event_subscriber_test.go`:
     * Instantiate two `EventSubscriber` instances sharing the same NATS server.
     * Publish messages to `ts.community.comm-1.agent.hub`.
     * Verify that messages are load-balanced between the subscribers (exactly one subscriber processes each message) rather than broadcast to both.

2. **GREEN Phase (Implement Minimum Code)**:
   * Update `cmd/agent/main.go` to parse the `TS_AGENT_ROLE` variable (via viper).
   * Update `EventSubscriber.Start` in `event_subscriber.go`:
     * If the role is `hub`, subscribe to `ts.community.{community_id}.agent.hub` using `nc.QueueSubscribe` with queue group `hub-queue-group`.
     * If the role is `spoke`, follow the standard subscription logic (subscribing to its own agent subject).

3. **REFACTOR Phase (Clean & Generalize)**:
   * Keep NATS subscription details encapsulated in the inbound adapter package.
   * Ensure tracing context and logging headers are properly populated for queue group deliveries.

## Acceptance Criteria

1. Event subscriber tests verify queue-group load balancing under the Hub role.
2. Multiple hub replicas can be run simultaneously without duplicate processing of identical thread events.
