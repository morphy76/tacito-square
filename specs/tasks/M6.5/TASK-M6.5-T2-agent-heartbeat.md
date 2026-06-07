# TASK-M6.5-T2: Agent Heartbeat Publisher

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.5-T2                          |
| Spec        | SPEC-FR-M6.5                          |
| Boundary    | Agent Heartbeat Publisher             |
| Status      | VERIFIED                              |
| Depends On  | TASK-M6.5-T1                          |

## Objective

Build the background loop on the Agent that aggregates runtime information (name, description, capabilities, active skills, tools) into the `AgentCard` and publishes it periodically as a heartbeat message over NATS with trace context propagation.

## Files

| File | Action |
|------|--------|
| `internal/agent/adapters/outbound/nats/heartbeat.go` | NEW |
| `internal/agent/adapters/outbound/nats/heartbeat_test.go` | NEW |
| `cmd/agent/main.go` | MODIFY |

## RED Phase

1. Create a test `TestHeartbeatPublisher_PublishesCard` in `internal/agent/adapters/outbound/nats/heartbeat_test.go` that spins up a test NATS server, registers a subscriber on `ts.community.test-comm.agent.test-agent.heartbeat`, initializes the publisher with dummy configurations, and asserts that a heartbeat event is published within the ticker period (mocking a fast tick time).
2. Verify that the test fails because `heartbeat.go` does not exist.

## GREEN Phase

1. Create `internal/agent/adapters/outbound/nats/heartbeat.go` implementing the heartbeat publishing loop.
2. In the publisher, compile the `AgentCard` details from configuration parameters, active brain engine settings, loaded skills, and registered MCP tools.
3. Start a ticker (default: 10s) that calls NATS to publish the `AgentCard` payload to the topic:
   `ts.community.{community_id}.agent.{agent_id}.heartbeat`
4. Populate event headers with `X-Tacito-Tenant` and inject active OpenTelemetry trace context.
5. In `cmd/agent/main.go`, initialize the heartbeat publisher service, run it in a background goroutine managed by a parent `context.Context` (preventing leaks), and register its lifecycle cleanup in the shutdown manager.
6. Verify that all tests compile and pass.

## REFACTOR Phase

- Ensure goroutines are cleanly terminated when context is cancelled.
- Verify that zerolog statements log startup/heartbeat errors gracefully without polluting stdout on normal runs.
