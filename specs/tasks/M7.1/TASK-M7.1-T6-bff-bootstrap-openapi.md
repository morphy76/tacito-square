# TASK-M7.1-T6: BFF Bootstrap, OpenAPI & Observability (`cmd/bff/`, `internal/bff/`, `internal/bff/bff_openapi.json`)

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Task ID     | TASK-M7.1-T6                                       |
| Spec        | SPEC-FR-M7.1                                       |
| Boundary    | BFF Wire-up — `cmd/bff/`, `internal/bff/`, observability |
| Status      | TODO                                               |
| Depends On  | TASK-M7.1-T4, TASK-M7.1-T5                         |

## Objective

Wire the BFF component end-to-end: update `bootstrap.go` with dependency injection, readiness probe dependencies, Prometheus metrics, OTel tracing, and zerolog structured logging. Produce a valid `GET /openapi.json` endpoint and the Swagger UI (dev mode only).

## Files

| File | Action |
|------|--------|
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |
| `internal/bff/bff_openapi.json` | NEW |
| `cmd/bff/main.go` | MODIFY |

## RED Phase

Extend `internal/bff/bootstrap_test.go`:

- `TestBFFServer_HealthzReturns200`: Assert `GET /healthz` returns `200 OK` without any dependency checks.
- `TestBFFServer_ReadyzReturns200_AllDepsHealthy`: Mock all dependency checkers (Redis ping, Keeper ping) as healthy; assert `GET /readyz` returns `200 OK` with a JSON body listing all checks as `ok`.
- `TestBFFServer_ReadyzReturns503_RedisFails`: Mock Redis ping failing; assert `GET /readyz` returns `503 Service Unavailable` with a JSON body naming `redis` as the failing dependency.
- `TestBFFServer_OpenAPIEndpoint`: Assert `GET /openapi.json` returns `200 OK` with `Content-Type: application/json` and a body containing `"openapi": "3.x"`.

Run `make test` — must fail (RED).

## GREEN Phase

1. **Update `internal/bff/bootstrap.go`** — `NewServer(cfg Config, sessionUC, eventUC, store, oidcProvider, keeperClient)`:
   - Initialize zerolog with JSON output to stdout; log component name and `VERSION.bff` at startup.
   - Initialize OTel OTLP gRPC tracer and wire W3C traceparent extraction middleware on all Gin routes.
   - Register Prometheus middleware (HTTP request count, latency histogram partitioned by route/method/status).
   - Register `/metrics` (Prometheus exposition).
   - Register `/healthz` — always `200 OK`, no dependency checks.
   - Register `/readyz` — parallel checks of Redis ping (`store.(Pinger)`) and Keeper ping (`keeperClient.Ping`); return `200` or `503` with per-dependency status.
   - Call `http.RegisterRoutes(r, sessionUC, eventUC)`.
   - Register `GET /openapi.json` serving the embedded `bff_openapi.json`.
   - Register `GET /swagger/*any` (Swagger UI) **only** when `GIN_MODE != release`.

2. **Create `internal/bff/bff_openapi.json`**:
   - Valid OpenAPI 3.x document (version `"3.1.0"`).
   - `info`: title `"BFF API"`, version from `VERSION.bff` placeholder (e.g., `"0.1.0"`).
   - Top-level `tags`: declare `auth/session`, `auth/backchannel`, `events/stream` with bounded context descriptions.
   - Paths:
     - `GET /api/bff/v1/auth/login` — tag `auth/session`
     - `GET /api/bff/v1/auth/callback` — tag `auth/session`
     - `POST /api/bff/v1/auth/logout` — tag `auth/session` (requires session cookie)
     - `POST /api/bff/v1/auth/backchannel-logout` — tag `auth/backchannel`
     - `GET /api/bff/v1/events/stream` — tag `events/stream`, response content-type `text/event-stream`
   - Include `components/securitySchemes` for cookie-based auth (`bff_session_id`).

3. **Update `cmd/bff/main.go`**:
   - Read all configuration from Viper (env vars + config file).
   - Construct concrete adapters (Redis store, OIDC client, Keeper client, SSE client).
   - Construct application services.
   - Call `bff.NewServer(...)`.
   - Start HTTP server with graceful shutdown on `SIGTERM`/`SIGINT`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm the OTel setup emits spans for all inbound Gin requests and all outbound calls (Redis, OIDC, Keeper HTTP).
- Confirm zerolog injects `trace_id` and `span_id` from the active OTel span context into every log entry.
- Confirm the Prometheus `/metrics` endpoint is NOT protected by the session middleware.
- Verify `GET /openapi.json` uses `go:embed` to embed the JSON file at build time (not runtime file reads).
- Confirm Swagger UI is registered only in non-release Gin mode.
