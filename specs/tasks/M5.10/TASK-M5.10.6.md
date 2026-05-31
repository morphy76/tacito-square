# TASK-M5.10.6: OpenTelemetry Tracing, Events & Exceptions Recording

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.6                                |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.2                                |

## Description

Instrument the active reasoning loop with OpenTelemetry (OTel), representing each step as a sub-span, recording cognitive thoughts/tool calls as Span Events, and explicitly recording exceptions and setting error statuses on active spans when errors occur.

## Work Items

1. **RED Phase**:
   - Write integration tests in `internal/agent/application/service/reasoning_otel_test.go` utilizing a mock OpenTelemetry exporter.
   - Assert that executing a reasoning loop creates an OTel sub-span linked to the parent context.
   - Assert that each thought and tool call registers a corresponding Span Event with attributes (`step_index`, `tool_name`).
   - Assert that a simulated tool error registers an exception event and sets the span status to `Error`.
   - Run the tests and assert failure (RED).

2. **GREEN Phase**:
   - Add OpenTelemetry trace context propagation and sub-span creation inside `cognitive_engine.go` and tool wrappers.
   - Call `span.AddEvent` to log thoughts and tool calls as span events with rich metadata attributes.
   - Call `span.RecordError` and `span.SetStatus(codes.Error, ...)` upon encountering errors in tool executions or loop steps.
   - Ensure the context with span is active across goroutines and channel boundaries.
   - Run tests and verify successful OTel instrumentation (GREEN).

3. **REFACTOR Phase**:
   - Inspect trace attributes to ensure clean metadata organization and absolute compliance with OpenTelemetry specifications.

## Acceptance Criteria

1. Sub-spans are created for each step in the reasoning cycle and correctly correlated with parent ingress spans.
2. Cognitive thoughts and tool invocations map to rich OpenTelemetry Span Events.
3. Failures in reasoning or tool execution set appropriate span error statuses and record exception parameters.
