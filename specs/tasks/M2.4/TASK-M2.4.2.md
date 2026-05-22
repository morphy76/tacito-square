# TASK-M2.4.2: Implement agent hello world (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.4.2                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.4                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.4.1                                 |

## Description

Implement agent hello world: Gin HTTP server for health probes only. All business logic arrives via NATS (later milestones).

## Work Items

1. Create `internal/agent/bootstrap.go`: `NewServer(logger, probe) *gin.Engine` — same pattern as keeper.
2. Create `cmd/agent/main.go`: config prefix `TS_AGENT`, default port `:8081`, health-only server.
3. Run `go test ./internal/agent/...` — all tests MUST pass.
4. Run `go build ./cmd/agent` — binary MUST build.

## Acceptance Criteria

1. Agent tests pass.
2. Binary builds and responds to `/healthz` and `/readyz`.
