# SPEC-FR-M2.6: BFF Hello World

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.6                                |
| Status        | IMPLEMENTED                                 |
| Milestone     | M2                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M2.2                                |
| Supersedes    | none                                        |

## Context

The BFF (Backend for Frontend) bridges keeper APIs to the Configurator and Auditor UIs. Before implementing API proxying (M7), it needs a working HTTP server with health probes, structured logging, and tracing. The BFF is conditionally deployed (`bff.enabled`).

## Specification

1. The `cmd/bff/main.go` entry point MUST initialize:
   - Viper configuration with `TS_BFF_*` environment variable prefix.
   - zerolog logger with JSON output.
   - OpenTelemetry tracer with OTLP exporter.
   - Gin HTTP server.
2. The Gin router MUST register:
   - `GET /healthz` — liveness probe (always 200).
   - `GET /readyz` — readiness probe (dependency check: keeper API reachability).
3. The server MUST listen on the port specified by `TS_BFF_PORT` (default: 8083).
4. The server MUST handle graceful shutdown on SIGTERM/SIGINT.
5. Startup MUST log component name (`bff`), version (from `VERSION.bff`), and listening port.

## Acceptance Criteria

1. `go build ./cmd/bff` produces a `bff` binary.
2. Running the binary starts an HTTP server on the configured port.
3. `GET /healthz` returns 200.
4. `GET /readyz` returns 200 when keeper is reachable, 503 otherwise.
5. Stdout contains valid JSON log lines with `component: "bff"`.
6. SIGTERM triggers graceful shutdown.

## Test Plan

1. **Build**: `go build -o bin/bff ./cmd/bff` — exit code 0.
2. **Health**: Start server with `gin.TestMode` + `httptest`, `GET /healthz` → 200.
3. **Readiness**: Mock keeper health check, `GET /readyz` → 200 / 503.
4. **Logging**: Capture stdout, parse JSON, assert fields.
5. **Shutdown**: Send SIGTERM, assert clean exit.

## Files Affected

- `cmd/bff/main.go` (MODIFY — bootstrap BFF HTTP server)
- `internal/bff/` (NEW — BFF-specific bootstrap)
