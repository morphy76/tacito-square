# TASK-M5.10.5: Logging & NATS Intermediate Reason Emission

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.5                                |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.2                                |

## Description

Implement structured `zerolog` stdout logging for each step of the cognitive loop and publish intermediate reasoning step events asynchronously to NATS, enabling consumers to stream reasoning evidence in real-time.

## Work Items

1. **RED Phase**:
   - Write integration tests in `internal/agent/adapters/inbound/nats/reasoning_subscriber_test.go` capturing NATS messages.
   - Assert that a mock execution of the reasoning loop publishes structured JSON messages to the subject `ts.tenant.{tenant_id}.agent.{agent_id}.thread.{thread_id}.reasoning` for each intermediate thought or action turn.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Add structured Zerolog entries inside `cognitive_engine.go` recording thoughts, actions, and observations using standard context keys (`tenant_id`, `agent_id`, `thread_id`, etc.).
   - Inject the NATS publisher port/client into the cognitive engine constructor.
   - For every iteration that yields a thought or tool call, publish the `AgentReasoningStepPayload` to the specified NATS subject asynchronously.
   - Run the tests to verify successful NATS emission and logging format compliance (GREEN).

3. **REFACTOR Phase**:
   - Inspect structured JSON logs to ensure no credentials or raw API keys leak in logs.
   - Ensure the NATS message structure contains tracing headers for correlation.

## Acceptance Criteria

1. Intermediate reasoning traces are logged to stdout in standardized JSON (`zerolog`) format.
2. The agent publishes intermediate step payloads over NATS to the designated reasoning subject before completing the user request with the final answer.
