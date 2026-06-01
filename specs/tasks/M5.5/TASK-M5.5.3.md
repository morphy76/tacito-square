# TASK-M5.5.3: Agent MCP Port and Resilient Adapter Implementation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.5.3                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M5.5                                |
| Depends On    | none                                        |

## Description

Define the hexagonal outbound port `ToolExecutor` inside the agent's application layer. Implement the concrete `mcp` adapter using the approved Model Context Protocol Go SDK (`github.com/modelcontextprotocol/go-sdk`). The adapter must support both `stdio` (subprocess execution) and `sse` (HTTP connection) transports, wrap outbound calls in circuit breakers and timeouts, manage subprocess lifetimes using Go context cancellations, and record structured telemetry (OTel spans and Prometheus metrics).

## Work Items

1. **RED Phase**:
   * Create the outbound port interface file `internal/agent/application/ports/outbound/tool_executor.go`.
   * Write unit tests in `internal/agent/adapters/outbound/mcp/mcp_adapter_test.go` asserting that:
     - Spawning standard MCP client adapters correctly resolves transport settings.
     - Timeout deadlines (`TS_AGENT_MCP_TIMEOUT_SECONDS`) are respected and cancel hanging executions.
     - Circuit breakers trip after 5 consecutive failures and return standard JSON fallback errors.
     - Calling `Close()` terminates spawned `stdio` subprocesses cleanly.
   * Verify test suite fails to compile or run as expected (RED).

2. **GREEN Phase**:
   * Implement the `ToolExecutor` port in `internal/agent/application/ports/outbound/tool_executor.go`.
   * Create `internal/agent/adapters/outbound/mcp/mcp_adapter.go` and implement the concrete MCP client manager using the approved Go SDK.
   * Wire `exec.CommandContext` using the context lifecycle to launch and pipe `stdio` subprocess streams, ensuring they terminate when the parent context completes.
   * Integrate the state-machine-based circuit breaker (`internal/agent/adapters/outbound/resiliency/circuit_breaker.go`) and standard context timeouts around all client operations.
   * Wrap executions in `mcp.execute` OpenTelemetry spans, recording metrics to standard gauges.
   * Verify all tests pass (GREEN).

3. **REFACTOR Phase**:
   * Inspect adapter package boundaries to guarantee absolutely zero leakages of external SDK internals or concrete connection properties back into the application layer.

## Acceptance Criteria

1. The `ToolExecutor` port isolates the cognitive loop from all raw connection and SDK structures.
2. The `mcp` adapter executes subprocess (`stdio`) and SSE commands securely, respecting the whitelisting rules, time limits, and circuit breaker trip thresholds.
3. Spawned subprocesses are cleanly killed on Close/cancellation, preventing process leaks.
