# BUG-M6.15: OrchestrationState Status Uses Raw Strings

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.15                                                          |
| Status        | OPEN                                                               |
| Severity      | LOW                                                                |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | [orchestration.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/domain/model/orchestration.go) |
| Violates      | `code_architecture` / Hexagonal DDD Boundaries                     |
| Discovered    | Code audit of the domain model structs.                            |

## Problem Statement

The `OrchestrationState` model in [orchestration.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/domain/model/orchestration.go) stores status as a raw string type:
```go
type OrchestrationState struct {
	...
	Status          string            `json:"status"`
	...
}
```
Throughout the orchestrator service, database/Redis adapters, and unit tests, status checks and assignments are performed using unstructured string literals like `"idle"`, `"waiting_spoke"`, etc. This lacks type safety and violates core Domain-Driven Design (DDD) principles where state attributes with predefined states should be represented as typed value objects or typed constants within the Domain layer.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Orchestration State Domain Model | [orchestration.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/domain/model/orchestration.go) | Raw `string` type used for `Status` field without typed constants or values. |

## Impact

1. **Typographical Bugs:** Easy to introduce subtle bugs if a string literal is misspelled (e.g., `"waiting-spoke"` vs `"waiting_spoke"`) in one of the adapter or service checks.
2. **Poor Design / Documentation:** Codebases lack a single source of truth for all valid states, making maintenance harder.

## Expected Behaviour

1. **Custom Type:** Define a dedicated type `OrchestrationStatus` of type `string` inside [orchestration.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/domain/model/orchestration.go).
2. **Typed Constants:** Declare explicit typed constants for all valid states:
   ```go
   type OrchestrationStatus string

   const (
       StatusIdle         OrchestrationStatus = "idle"
       StatusWaitingSpoke OrchestrationStatus = "waiting_spoke"
   )
   ```
3. **Model Integration:** Update `OrchestrationState.Status` to use the `OrchestrationStatus` type, and refactor existing references to use these constants.

## Acceptance Criteria

1. `OrchestrationState.Status` is declared with a typed custom type rather than a raw string.
2. Status-related evaluations throughout the agent component are refactored to use the new typed constants.
3. Compilation succeeds and all tests pass with the refactored typed status fields.
