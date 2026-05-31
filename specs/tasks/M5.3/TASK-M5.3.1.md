# TASK-M5.3.1: Short-Term Memory Domain Models and Outbound Port Interface

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.3.1                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.3                                |
| Depends On    | none                                        |

## Description

Define pure, stateless domain models for Short-Term Memory (STM) entries in the agent domain layer and declare the `ShortTermMemory` outbound port interface in the application ports layer to govern outbound Redis memory interactions. Extend the existing `BrainRequest` model to accept a history list of memory entries.

## Work Items

1. **RED Phase**:
   - Write a unit test `internal/agent/domain/model/memory_test.go` asserting JSON serialization/deserialization validation rules for memory entries.
   - Write a unit test in `internal/agent/application/ports/outbound/memory_test.go` asserting compilation compliance of mock outbound memory handlers against the `ShortTermMemory` port interface.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Create `internal/agent/domain/model/memory.go` with the `MemoryEntry` struct and validation helper methods.
   - Modify `internal/agent/domain/model/brain.go` to include `History []MemoryEntry` in `BrainRequest`.
   - Create `internal/agent/application/ports/outbound/memory.go` declaring the `ShortTermMemory` interface with `Append`, `Get`, and `Clear` methods.
   - Run the tests to verify compilation and successful execution (GREEN).

3. **REFACTOR Phase**:
   - Inspect imports to ensure `internal/agent/domain/` maintains absolute code purity (no dependencies on `application/` or `adapters/` packages).

## Acceptance Criteria

1. Hexagonal boundary check: all domain structures and interface signatures compile successfully with zero outbound package imports.
2. The outbound `ShortTermMemory` interface correctly structures multi-tenant lookup coordinates (`tenantID`, `agentID`, `threadID`) and sliding window parameters (`limit`).
