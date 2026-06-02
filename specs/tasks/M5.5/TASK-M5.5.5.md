# TASK-M5.5.5: Main Bootstrap and Dependency Injection Wiring

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.5.5                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.5                                |
| Depends On    | TASK-M5.5.3, TASK-M5.5.4                    |

## Description

Wire the concrete `mcp` adapter and `ToolExecutor` outbound port into the agent's main application bootstrap inside `cmd/agent/main.go`. Configure environment variables and Viper defaults, register the active client connections with the graceful shutdown manager (`mgr`) to trigger process/transport cleanups on pod terminations, and append downstream SSE MCP client ping checks to the readiness health probe registry (`/readyz`).

## Work Items

1. **RED Phase**:
   * Write a bootstrap test inside `internal/agent/bootstrap_test.go` or a mock main execution run asserting that loading a mock environment containing `TS_AGENT_MCP_CLIENTS` successfully binds configuration values to Viper and executes without runtime DI errors.
   * Run the test suite and verify it fails (RED) because `main.go` has no references to the new `mcp` adapter or `ToolExecutor` ports.

2. **GREEN Phase**:
   * Modify `cmd/agent/main.go` to add standard Viper config mappings and defaults for `mcp.clients` (`TS_AGENT_MCP_CLIENTS`) and `TS_AGENT_MCP_TIMEOUT_SECONDS`.
   * In `main()`, retrieve the environment config, instantiate the concrete `mcp` adapter, and register its `Close(ctx)` method with the shutdown manager `mgr` under the key `"mcp-client-executor"`.
   * Pass the instantiated `mcp` adapter into the `CognitiveEngine` constructor or wire it via a fluent config method (e.g. `WithToolExecutor`).
   * Update the parallel readiness probe (`/readyz`) checkers slice inside `main.go` to perform ping connection checks targeting any configured SSE MCP client URLs in parallel with database, NATS, and cache checks.
   * Verify all tests and dry-run execution pipelines pass successfully (GREEN).

3. **REFACTOR Phase**:
   * Inspect the `main.go` dependency graph to verify that the application layer is exclusively dependent on the `ToolExecutor` interface port rather than any concrete MCP client classes or SDK parameters.

## Acceptance Criteria

1. The agent boots cleanly with valid `TS_AGENT_MCP_CLIENTS` env values, parsing them and configuring all active transports.
2. Graceful shutdown signals (`SIGTERM`, `SIGINT`) trigger `Close()` on the tool executor, cleanly terminating all child `stdio` processes and SSE client connections.
3. Configured SSE MCP clients are checked in parallel during readiness probes (`GET /readyz`).
