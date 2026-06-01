# TASK-M5.5.4: Cognitive Engine Tool Registry and Whitelist Enforcement

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.5.4                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.5                                |
| Depends On    | TASK-M5.5.3                                 |

## Description

Integrate the `ToolExecutor` outbound port into the `CognitiveEngine` reasoning loop. Parse `TS_AGENT_MCP_CLIENTS` at startup. Discovered external tools must be strictly filtered against their client's `allowed_tools` whitelist (with a default-deny on empty whitelists) before being exposed to the LLM. Built-in tools like `enable_skill` and `recall_memory` must remain exempt from environment-driven checks. Map incoming ReAct tool calls to the active registry and route them to the `ToolExecutor` port, and export custom Prometheus metrics for tool executions.

## Work Items

1. **RED Phase**:
   * Write unit and integration tests inside `internal/agent/application/service/cognitive_engine_test.go` asserting that:
     - Discovered tools NOT present in `allowed_tools` are filtered out and never exposed to the LLM.
     - If `allowed_tools` is empty, no tools from that client are exposed (deny-all default).
     - Tool calls to unlisted tools are rejected with immediate validation observations.
     - Built-in tools (`enable_skill`, `recall_memory`) continue to function and execute normally.
     - Prometheus metrics endpoint `GET /metrics` exports request count and duration histogram data for MCP invocations.
   * Verify test suite fails to compile or run as expected (RED).

2. **GREEN Phase**:
   * Update `internal/agent/application/service/cognitive_engine.go` to accept the `ToolExecutor` outbound port.
   * Implement parsing of `TS_AGENT_MCP_CLIENTS` during engine bootstrap.
   * Intersect discovered client schemas with their configured whitelists, registering whitelisted items into the thread-local active tools registry.
   * Enforce default-deny and tool execution mapping check inside the ReAct step execution flow.
   * Instrument the tool execution routing to increment `ts_agent_mcp_requests_total` and log duration statistics.
   * Verify all tests compile and pass successfully (GREEN).

3. **REFACTOR Phase**:
   * Verify that no external library specific properties leak into `cognitive_engine.go`, keeping the domain loop purely abstract.

## Acceptance Criteria

1. Whitelist enforcement: only tools explicitly listed in `allowed_tools` are exposed to the LLM and executed. An empty list results in a deny-all state.
2. Built-in cognitive tools are exempt from environment whitelists.
3. The reasoning engine publishes step telemetry and logs step telemetry correctly with correct OTel integration.
