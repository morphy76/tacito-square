# TASK-REFACTOR-memory_adapter: Refactor memory_adapter.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-memory_adapter                |
| Status        | VERIFIED                                    |
| Target File   | [memory_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/memory_adapter.go)  |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Split OrchestrationStateStore and ThreadLock implementation functions out of `memory_adapter.go` into a new file `state_lock_adapter.go` to reduce file length and complexity.

## Work Items
1. **Baseline Phase**:
   - [x] Verify all existing tests pass.
2. **Refactor Phase**:
   - [x] Create a new file [state_lock_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/state_lock_adapter.go) and move `formatStateKey`, `formatLockKey`, `SaveState`, `GetState`, `ClearState`, `Lock`, and `Unlock` to it.
   - [x] Keep `NewRedisMemoryAdapter`, `Close`, `Ping`, `formatKey`, `Append`, `Get`, `Clear`, and `RollbackLast` inside [memory_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/memory_adapter.go).
3. **Verification Phase**:
   - [x] Run existing integration tests to ensure they are 100% green.
   - [x] Run `make lint` to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. [memory_adapter.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/adapters/outbound/redis/memory_adapter.go) has its LOC reduced from 610 lines to under 400 lines.
3. All existing agent and orchestrator tests remain fully green.
4. Lint checks pass cleanly.
