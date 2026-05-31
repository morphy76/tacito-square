# TASK-M5.4.1: Long-Term Memory Domain Models and Outbound Interfaces

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.4.1                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.4                                |
| Depends On    | none                                        |

## Description

Define pure, stateless domain models for Long-Term Memory (LTM) entries, including classification types and visibility search filters. Declare the outbound `LongTermMemory` and `Embedder` ports in the application layer to govern outbound Qdrant vector interactions and text-embedding generation.

## Work Items

1. **RED Phase**:
   - Write a unit test `internal/agent/domain/model/ltm_test.go` asserting JSON serialization, deserialization, and field validation rules for long-term memory entries and filters.
   - Write a unit test in `internal/agent/application/ports/outbound/ltm_test.go` asserting compilation compliance of mock outbound handlers against `LongTermMemory` and `Embedder` port interfaces.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Create `internal/agent/domain/model/ltm.go` containing `LTMEntryType` constants, the `LTMEntry` struct with its validation method, and the `LTMFilter` struct.
   - Create `internal/agent/application/ports/outbound/embedder.go` declaring the `Embedder` interface with `CreateEmbedding` and `CreateEmbeddingsBatch` signatures.
   - Create `internal/agent/application/ports/outbound/ltm.go` declaring the `LongTermMemory` interface with `Save`, `Search`, and `Delete` signatures.
   - Run the tests to verify compilation and successful execution (GREEN).

3. **REFACTOR Phase**:
   - Inspect imports to ensure `internal/agent/domain/` maintains absolute code purity (no dependencies on `application/` or `adapters/` packages).

## Acceptance Criteria

1. Hexagonal boundary check: all domain structures and interface signatures compile successfully with zero outbound package imports.
2. The outbound `LongTermMemory` interface correctly structures multi-tenant coordinate boundaries (`tenantID`, `agentID`) and visibility scoping rules (`LTMFilter`).
