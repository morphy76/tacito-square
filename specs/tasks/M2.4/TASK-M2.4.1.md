# TASK-M2.4.1: Agent bootstrap tests (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.4.1                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.4                                |
| Phase         | RED                                         |
| Depends On    | TASK-M2.2.2                                 |

## Description

Write tests defining the agent's hello-world behavior. The agent is an HTTP-for-health-only server (per user decision: HTTP stack just for health, everything else via NATS in later milestones).

## Work Items

1. Create `internal/agent/bootstrap_test.go` with:
   - `TestNewServer_ReturnsGinEngine` — verifies `NewServer()` returns a configured `*gin.Engine`.
   - `TestHealthz_Returns200` — GET `/healthz` returns 200 with `{"status":"alive"}`.
   - `TestReadyz_Returns200` — GET `/readyz` returns 200 (no dependency checkers for hello world).
2. Run `go test ./internal/agent/...` — tests MUST FAIL.

## Acceptance Criteria

1. `internal/agent/bootstrap_test.go` exists.
2. Tests FAIL because package doesn't exist.
