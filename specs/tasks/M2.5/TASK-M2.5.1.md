# TASK-M2.5.1: Operator bootstrap tests (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.5.1                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.5                                |
| Phase         | RED                                         |
| Depends On    | TASK-M2.2.2                                 |

## Description

Write tests defining the operator's hello-world behavior. The operator uses Gin for health probes (like the other components). Controller-runtime manager integration comes in M4 — for now it's just a health-only server.

## Work Items

1. Create `internal/operator/bootstrap_test.go` with:
   - `TestNewServer_ReturnsGinEngine` — verifies `NewServer()` returns a configured `*gin.Engine`.
   - `TestHealthz_Returns200` — GET `/healthz` returns 200.
   - `TestReadyz_Returns200` — GET `/readyz` returns 200.
2. Run `go test ./internal/operator/...` — tests MUST FAIL.

## Acceptance Criteria

1. `internal/operator/bootstrap_test.go` exists.
2. Tests FAIL because package doesn't exist.
