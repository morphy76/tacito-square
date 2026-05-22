# TASK-M2.6.1: BFF bootstrap tests (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.6.1                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.6                                |
| Phase         | RED                                         |
| Depends On    | TASK-M2.2.2                                 |

## Description

Write tests defining the BFF's hello-world behavior. The BFF bridges keeper APIs to the UIs — for now it's a health-only Gin server.

## Work Items

1. Create `internal/bff/bootstrap_test.go` with:
   - `TestNewServer_ReturnsGinEngine` — verifies `NewServer()` returns a configured `*gin.Engine`.
   - `TestHealthz_Returns200` — GET `/healthz` returns 200.
   - `TestReadyz_Returns200` — GET `/readyz` returns 200.
2. Run `go test ./internal/bff/...` — tests MUST FAIL.

## Acceptance Criteria

1. `internal/bff/bootstrap_test.go` exists.
2. Tests FAIL because package doesn't exist.
