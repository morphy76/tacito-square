# TASK-M2.6.2: Implement BFF hello world (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.6.2                                 |
| Status        | DONE                                        |
| Spec          | SPEC-FR-M2.6                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.6.1                                 |

## Description

Implement BFF hello world: Gin HTTP server for health probes only. API bridge to keeper deferred to M7.

## Work Items

1. Create `internal/bff/bootstrap.go`: `NewServer(logger, probe) *gin.Engine`.
2. Create `cmd/bff/main.go`: config prefix `TS_BFF`, default port `:8083`.
3. Run `go test ./internal/bff/...` — all tests MUST pass.
4. Run `go build ./cmd/bff`.

## Acceptance Criteria

1. BFF tests pass. Binary builds.
