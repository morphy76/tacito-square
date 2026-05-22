# SPEC-FR-M2.2: Shared Foundation Library

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.2                                |
| Status        | IMPLEMENTED                                       |
| Milestone     | M2                                          |
| Component     | shared                                      |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

All 4 components share common cross-cutting concerns: configuration loading, health probes, structured logging, distributed tracing, error handling, and graceful shutdown. These are provided by a shared library in `internal/shared/` to ensure consistency and reduce duplication.

## Specification

1. **Config** (`internal/shared/config/`): The system MUST provide Viper-based configuration loading from environment variables with the `TS_<COMPONENT>_*` prefix convention (per SPEC-NFR-STACK).
2. **Health** (`internal/shared/health/`): The system MUST provide Gin handlers for `/healthz` (liveness — always 200) and `/readyz` (readiness — parallel dependency checks with 5s timeout, 503 on failure) (per SPEC-NFR-HEALTH).
3. **Observability** (`internal/shared/observability/`):
   - Logging MUST use zerolog with JSON output to stdout (per SPEC-NFR-LOG).
   - Tracing MUST use OpenTelemetry SDK with OTLP gRPC exporter (per SPEC-NFR-STACK).
   - Every log entry MUST include `trace_id` and `span_id` from the active OTel context.
   - Component name and version MUST be logged at startup.
4. **Errors** (`internal/shared/errors/`): The system MUST provide standard error types and a Gin error handler that produces the JSON error format defined in SPEC-NFR-HTTP.
5. **Ports** (`internal/shared/ports/`): The system MUST define shared outbound port interfaces (e.g., `HealthChecker`) following hexagonal architecture (per SPEC-NFR-HEXAGONAL).
6. **Graceful Shutdown**: The system MUST provide an `os.Signal`-aware shutdown helper that drains HTTP connections and flushes telemetry before exit.

## Acceptance Criteria

1. Configuration loads from environment variables and fails fast on missing required values.
2. `/healthz` returns 200 with `{"status": "ok"}`.
3. `/readyz` returns 200 when all dependencies are reachable, 503 with per-dependency status otherwise.
4. Log output is valid JSON with `trace_id`, `span_id`, `level`, `message`, `component`, `version` fields.
5. OTel traces are exported to the configured collector endpoint.
6. Standard error responses follow `{"error": {"code": N, "message": "..."}}` format.
7. Graceful shutdown completes within 30 seconds on SIGTERM.

## Test Plan

1. **Config**: Unit test — set env vars, load config, assert values. Test missing required var → panic/fatal.
2. **Health liveness**: Unit test — `GET /healthz` → 200, body `{"status": "ok"}`.
3. **Health readiness**: Unit test with mock `HealthChecker` — all pass → 200; one fails → 503 with status map.
4. **Logging**: Unit test — capture log output, parse JSON, assert required fields present.
5. **Errors**: Unit test — trigger error handler, assert JSON error format.
6. **Tracing**: Integration test with in-memory exporter — verify span creation and propagation.

## Files Affected

- `internal/shared/config/` (MODIFY — ensure Viper env loading)
- `internal/shared/health/` (MODIFY — ensure liveness + readiness handlers)
- `internal/shared/observability/` (MODIFY — ensure zerolog + OTel setup)
- `internal/shared/errors/` (MODIFY — ensure standard error types)
- `internal/shared/ports/` (MODIFY — ensure HealthChecker port)
- `internal/shared/shutdown.go` (NEW — graceful shutdown helper)
