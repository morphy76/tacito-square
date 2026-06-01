# TASK-M5.10.3: Recall Memory Cognitive Tool Wrapper

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.3                                |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.2                                |

## Description

Implement the `recall_memory` built-in cognitive tool to allow the active reasoning loop to execute semantic long-term memory lookups via the outbound `LongTermMemory` and `Embedder` ports.

## Work Items

1. **RED Phase**:
   - Write integration tests in `internal/agent/application/service/recall_tool_test.go` utilizing mock `Embedder` and `LongTermMemory` ports.
   - Verify that calling `recall_memory` successfully propagates mandatory `tenantID` and visibility filtering contexts.
   - Run tests and verify failure (RED).

2. **GREEN Phase**:
   - Register the `recall_memory` tool schema into the cognitive engine's tool list.
   - Implement the execution block for the `recall_memory` tool inside `cognitive_engine.go`.
   - Wire tool invocation to call `Embedder.CreateEmbedding` and `LongTermMemory.Search`.
   - Implement graceful fallback: if the ports return an error, catch it, log a warning, and return a standard JSON error observation to the LLM.
   - Run tests and verify successful completion (GREEN).

3. **REFACTOR Phase**:
   - Clean up mapping between LTM entry properties and the tool's JSON string observation formatting.

## Acceptance Criteria

1. Vector memory database lookups are triggered only when the LLM explicitly executes the `recall_memory` tool.
2. The search query strictly maps standard multi-tenant scopes and community boundaries.
3. Failures in downstream vector infrastructure return clean JSON fallback notifications to the reasoning loop.
