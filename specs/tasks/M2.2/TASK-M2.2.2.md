# TASK-M2.2.2: Implement graceful shutdown package (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.2.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.2                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.2.1                                 |

## Description

Implement the `internal/shared/shutdown` package. This is the only missing shared package — config, health, observability, and errors already exist and have tests.

## Work Items

1. Create `internal/shared/shutdown/shutdown.go`:
   - `type Hook func(ctx context.Context) error` — a cleanup function.
   - `type Manager` struct — registers hooks and runs them on shutdown.
   - `NewManager(timeout time.Duration) *Manager` — creates manager with shutdown timeout.
   - `Register(name string, hook Hook)` — adds a shutdown hook.
   - `Wait(signals ...os.Signal)` — blocks until signal received, then runs hooks in reverse order with timeout context.
   - Log each hook execution and any errors via zerolog.
2. Run `go test ./internal/shared/shutdown/...` — tests from TASK-M2.2.1 MUST pass.

## Acceptance Criteria

1. `internal/shared/shutdown/shutdown.go` implements the interface defined by tests.
2. All shutdown tests pass.
3. Existing tests in other packages remain green.
