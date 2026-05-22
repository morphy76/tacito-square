# SPEC-FR-M2.3: Keeper Hello World

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.3                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M2                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.2                                |
| Supersedes    | none                                        |

## Context

The keeper is the control plane for Tacito Square. Before implementing domain logic (M3), it needs a working hello-world HTTP server with health probes, structured logging, tracing, and configuration loading. This establishes the foundation for incremental feature builds.

## Specification

1. The `cmd/keeper/main.go` entry point MUST initialize:
   - Viper configuration with `TS_KEEPER_*` environment variable prefix.
   - zerolog logger with JSON output.
   - OpenTelemetry tracer with OTLP exporter.
   - Gin HTTP server in release mode.
2. The Gin router MUST register:
   - `GET /healthz` — liveness probe (always 200).
   - `GET /readyz` — readiness probe (dependency checks: PostgreSQL, NATS, Redis).
3. The server MUST listen on the port specified by `TS_KEEPER_PORT` (default: 8080).
4. The server MUST handle graceful shutdown on SIGTERM/SIGINT.
5. Startup MUST log component name (`keeper`), version (from `VERSION.keeper`), and listening port.

## Acceptance Criteria

1. `go build ./cmd/keeper` produces a `keeper` binary.
2. Running the binary starts an HTTP server on the configured port.
3. `GET /healthz` returns 200.
4. `GET /readyz` returns 200 when dependencies are mocked/available, 503 otherwise.
5. Stdout contains valid JSON log lines with `component: "keeper"`.
6. SIGTERM triggers graceful shutdown within 30 seconds.

## Test Plan

1. **Build**: `go build -o bin/keeper ./cmd/keeper` — exit code 0.
2. **Health**: Start server with `gin.TestMode` + `httptest`, `GET /healthz` → 200.
3. **Readiness**: Mock dependency checkers, `GET /readyz` → 200 (all pass), 503 (one fails).
4. **Logging**: Capture stdout, parse JSON, assert `component`, `version`, `level` fields.
5. **Shutdown**: Send SIGTERM, assert server stops within 30s.

## Files Affected

- `cmd/keeper/main.go` (MODIFY — bootstrap keeper HTTP server)
- `internal/keeper/` (NEW — keeper-specific application bootstrap)
