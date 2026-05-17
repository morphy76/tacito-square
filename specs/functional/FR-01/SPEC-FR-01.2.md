# SPEC-FR-01.2: Agent State Transitions

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-01.2                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-01.2                            |
| Component     | keeper                             |
| Depends On    | —                                  |

## Context

The Keeper tracks agent lifecycle through a finite state machine. Invalid transitions MUST be rejected at the domain level.

## Specification

1. `AgentInstance.Status` MUST be one of: `Pending`, `Running`, `Degraded`, `Terminated`.
2. Valid transitions:
   - `Pending → Running` (deploy confirmed)
   - `Running → Degraded` (heartbeat timeout)
   - `Running → Terminated` (user or rule driven)
   - `Degraded → Running` (recovery)
   - `Degraded → Terminated` (escalation)
3. Invalid transitions MUST return a domain error (e.g., `Terminated → Running`).
4. `Pending → Degraded` is NOT a valid transition.

## Acceptance Criteria

1. `TransitionTo(Running)` from `Pending` succeeds
2. `TransitionTo(Degraded)` from `Running` succeeds
3. `TransitionTo(Terminated)` from `Running` succeeds
4. `TransitionTo(Running)` from `Terminated` returns error
5. `TransitionTo(Degraded)` from `Pending` returns error

## Test Plan

- Unit tests in `internal/keeper/domain/domain_test.go`
- 9 test cases covering all valid and invalid transitions

## Files

- `internal/keeper/domain/agent_instance.go` ✅ IMPLEMENTED
- `internal/keeper/domain/domain_test.go` ✅ 9 tests passing
