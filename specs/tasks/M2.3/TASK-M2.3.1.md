# TASK-M2.3.1: Keeper bootstrap tests (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.3.1                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.3                                |
| Phase         | RED                                         |
| Depends On    | TASK-M2.2.2                                 |

## Description

Write tests that define the keeper's hello-world behavior: a Gin HTTP server bootstrapping with health probes, structured logging, and OTel tracing. Tests exercise the bootstrap function, not `main()` directly.

## Work Items

1. Create `internal/keeper/bootstrap_test.go` with:
   - `TestNewServer_ReturnsGinEngine` — verifies `NewServer()` returns a configured `*gin.Engine`.
   - `TestHealthz_Returns200` — GET `/healthz` returns 200 with `{"status":"alive"}`.
   - `TestReadyz_Returns200` — GET `/readyz` returns 200 with `{"status":"ready"}` (no dependency checkers in hello world).
   - `TestServer_LogsOnStartup` — verifies structured log output contains component name.
2. Run `go test ./internal/keeper/...` — tests MUST FAIL (package doesn't exist yet).

## Acceptance Criteria

1. `internal/keeper/bootstrap_test.go` exists.
2. Tests define the hello-world bootstrap interface.
3. Tests FAIL because `internal/keeper/` package doesn't exist.
