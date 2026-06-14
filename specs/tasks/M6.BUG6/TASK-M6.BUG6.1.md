# TASK-M6.BUG6.1: Graceful Orchestration Loop Limit & Directives

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.BUG6.1                              |
| Status        | IMPLEMENTED                                 |
| Spec          | BUG-M6.6                                    |
| Depends On    | none                                        |

## Description

Refine prompt directives in `internal/agent/application/service/orchestrator.go` to explicitly guide the Hub agent to yield to the user when clarifying information is requested by a spoke. Additionally, update the loop limit fallback logic in the orchestrator so it does not terminate the thread (preserves spoke memory) and returns the latest spoke response to the user.

## Scope

This task covers a single logical boundary: **Agent Orchestrator Service** (`orchestrator.go`) and its unit tests (`orchestrator_test.go`).

## Work Items

1. **RED Phase**:
   - In `internal/agent/application/service/orchestrator_test.go`, update unit tests to verify:
     - Reaching loop limit does not publish `EndThread` event.
     - Reaching loop limit returns the latest spoke response (`"Spoke response"`) instead of `"Orchestration limit exceeded"`.
     - Assertion of `publishes` length and content.
   - Run tests and see them fail (RED).

2. **GREEN Phase**:
   - Implement the system prompt directives changes in `internal/agent/application/service/orchestrator.go`.
   - Implement the loop limit fallback changes in `internal/agent/application/service/orchestrator.go`.
   - Run tests and see them pass (GREEN).

3. **REFACTOR Phase**:
   - Clean up code, check formatting, and ensure complete test coverage.
