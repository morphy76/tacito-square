# TASK-M6.2.1: NATS JetStream Infrastructure Config & Outbound Publishers

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.2.1                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M6.2                                |
| Depends On    | none                                        |

## Description

Enable NATS JetStream in the infrastructure Helm values and update the NATS Event Publishers in both the agent and keeper components to publish to JetStream with server-side acknowledgements, mapping the event ID header.

## Work Items

1. **RED Phase**:
   - Create/update tests in `internal/keeper/adapters/outbound/nats/event_publisher_test.go` and `internal/agent/adapters/outbound/nats/publisher_test.go` to:
     - Verify publishing to standard pub/sub subjects does NOT block or use JetStream.
     - Verify publishing to JetStream stream subjects (`ts.community.>`) uses JetStream context and awaits publish acknowledgement.
     - Verify `Nats-Msg-Id` is correctly populated with the event ID.
     - Ensure these tests fail initially on the existing codebase (or fail to compile/run due to missing JetStream mocks/support).

2. **GREEN Phase**:
   - Modify `tools/helm/tacito-square-infra/values.yaml` to set `nats.config.jetstream.enabled: true`.
   - Update `internal/keeper/adapters/outbound/nats/event_publisher.go`:
     - Access JetStream context via `nc.JetStream()` or initialize it on setup.
     - If the subject matches `ts.community.>`, publish using JetStream `PublishMsg` with context timeout propagation.
     - Set the `Nats-Msg-Id` header to match `event.EventID`.
     - Otherwise, publish using standard `nc.PublishMsg` for non-stream events.
   - Update `internal/agent/adapters/outbound/nats/publisher.go`:
     - Implement the same publishing behavior for the agent's event publisher.

3. **REFACTOR Phase**:
   - Ensure clean abstraction of JetStream context initialization.
   - Guard against initialization failure if JetStream is disabled in tests by dynamically checking or allowing fallback (or enabling it in test suite).
   - Ensure error messages are wrapped properly according to specifications.

## Acceptance Criteria

1. `tools/helm/tacito-square-infra/values.yaml` has `nats.config.jetstream.enabled: true`.
2. Outbound publishers successfully publish to JetStream streams with blocking ACKs and headers.
3. Unit and integration tests for publishers compile and pass.
