# TASK-M5.10.1: Cognitive Loop Domain Models and Ports

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.1                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | none                                        |

## Description

Define pure, stateless domain models representing active reasoning loop traces, intermediate thoughts, cognitive tool invocation schemas, and dynamic skill status blocks.

## Work Items

1. **RED Phase**:
   - Write a unit test `internal/agent/domain/model/reasoning_test.go` asserting JSON serialization, deserialization, and validation of intermediate reasoning steps.
   - Assert that invalid reasoning step payloads (e.g. negative step index or missing timestamp) trigger validation errors.
   - Run the tests and verify failure (RED).

2. **GREEN Phase**:
   - Create `internal/agent/domain/model/reasoning.go` containing `AgentReasoningStepPayload` (fields: `step_index`, `thought`, `action`, `observation`, `timestamp`).
   - Define a pure `Validate()` method for reasoning payloads.
   - Run the tests and assert successful compilation and validation (GREEN).

3. **REFACTOR Phase**:
   - Inspect imports to verify that `internal/agent/domain/` maintains absolute code purity with zero external library or framework dependencies.

## Acceptance Criteria

1. Hexagonal boundary check: all reasoning model structures compile with zero application or adapter package imports.
2. The reasoning trace model supports standard JSON serialization to integrate with structured log streams and NATS payloads.
