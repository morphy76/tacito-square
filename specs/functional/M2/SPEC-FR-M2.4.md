# SPEC-FR-M2.4: Agent Hello World

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.4                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M2                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M2.2                                |
| Supersedes    | none                                        |

## Context

The agent is a zero-scalable runtime that reasons via LLM, uses tools, and communicates via NATS. Before implementing domain logic (M5), it needs a minimal HTTP server for health probes only, plus structured logging and tracing. The agent's primary communication channel is NATS, not HTTP.

## Specification

1. The `cmd/agent/main.go` entry point MUST initialize:
   - Viper configuration with `TS_AGENT_*` environment variable prefix.
   - zerolog logger with JSON output.
   - OpenTelemetry tracer with OTLP exporter.
   - Gin HTTP server (health probes only — no REST API).
2. The Gin router MUST register:
   - `GET /healthz` — liveness probe (always 200).
   - `GET /readyz` — readiness probe (dependency checks: NATS, Redis, Qdrant).
3. The server MUST listen on the port specified by `TS_AGENT_PORT` (default: 8081).
4. The server MUST handle graceful shutdown on SIGTERM/SIGINT.
5. Startup MUST log component name (`agent`), version (from `VERSION.agent`), agent ID, and community ID.

## Acceptance Criteria

1. `go build ./cmd/agent` produces an `agent` binary.
2. Running the binary starts an HTTP server on the configured port.
3. `GET /healthz` returns 200.
4. `GET /readyz` returns 200 when dependencies are mocked/available, 503 otherwise.
5. Stdout contains valid JSON log lines with `component: "agent"`.
6. SIGTERM triggers graceful shutdown.

## Test Plan

1. **Build**: `go build -o bin/agent ./cmd/agent` — exit code 0.
2. **Health**: Start server with `gin.TestMode` + `httptest`, `GET /healthz` → 200.
3. **Readiness**: Mock dependency checkers, `GET /readyz` → 200 / 503.
4. **Logging**: Capture stdout, parse JSON, assert fields.
5. **Shutdown**: Send SIGTERM, assert clean exit.

## Files Affected

- `cmd/agent/main.go` (MODIFY — bootstrap agent HTTP server)
- `internal/agent/` (NEW — agent-specific application bootstrap)
