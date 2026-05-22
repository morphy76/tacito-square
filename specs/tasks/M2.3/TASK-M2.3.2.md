# TASK-M2.3.2: Implement keeper hello world (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.3.2                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.3                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.3.1                                 |

## Description

Implement the keeper hello-world: a Gin HTTP server with health probes, structured logging, OTel tracing, and graceful shutdown. This is the minimum to make the RED tests pass.

## Work Items

1. Create `internal/keeper/bootstrap.go`:
   - `NewServer(logger zerolog.Logger, probe *health.Probe) *gin.Engine` — creates Gin engine, registers `/healthz` and `/readyz` routes.
   - Use `gin.New()` with recovery middleware and zerolog-based request logging.
2. Create `cmd/keeper/main.go`:
   - Load config via `config.Load("TS_KEEPER")`.
   - Initialize logger via `observability.NewLogger(...)`.
   - Initialize tracer via `observability.InitTracer(...)`.
   - Create health probe (no dependency checkers for hello world).
   - Create and start Gin server on `TS_KEEPER_PORT` (default `:8080`).
   - Use `shutdown.Manager` for graceful shutdown (tracer shutdown, server shutdown).
3. Run `go test ./internal/keeper/...` — all tests MUST pass.
4. Run `go build ./cmd/keeper` — binary MUST build.

## Acceptance Criteria

1. `internal/keeper/bootstrap.go` exists with `NewServer()`.
2. `cmd/keeper/main.go` exists and builds.
3. All keeper tests pass.
4. Binary starts, responds to `/healthz` and `/readyz`, and shuts down gracefully on SIGTERM.
