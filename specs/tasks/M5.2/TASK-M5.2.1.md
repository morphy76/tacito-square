# TASK-M5.2.1: Domain Models and Brain Outbound Port

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.2.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.2                                |
| Depends On    | none                                        |

## Description

Define pure, stateless domain models for LLM reasoning in the agent domain layer and declare the `Brain` outbound port interface in the application ports layer to govern outbound LLM interactions.

## Work Items

1. **RED Phase**:
   - Write a unit test `internal/agent/domain/model/brain_test.go` that asserts request options and parameters validation.
   - Write a unit test in `internal/agent/application/ports/outbound/brain_test.go` asserting compile compliance of mock outbound handlers against the `Brain` interface contract.
   - Run tests and witness expected compilation or implementation failure (RED).

2. **GREEN Phase**:
   - Implement stateless models inside `internal/agent/domain/model/brain.go`:
     - `BrainRequest`: containing system prompt, user prompt, and LLM hyperparameters.
     - `BrainResponse`: containing text content, usage statistics, and completion markers.
     - `BrainStreamChunk`: containing partial content fragments.
   - Define the `Brain` interface inside `internal/agent/application/ports/outbound/brain.go`.
   - Verify tests pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Inspect import scopes to ensure `internal/agent/domain/` contains zero external application layer packages, guaranteeing absolute domain layer purity.

## Acceptance Criteria

1. Hexagonal boundary check: all domain structures compile successfully with zero outbound package imports.
2. The outbound `Brain` interface correctly models standard text generation and stream generation functions.
