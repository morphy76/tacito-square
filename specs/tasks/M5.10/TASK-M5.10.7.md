# TASK-M5.10.7: Integrate Cognitive Engine in Message Processor Orchestrator

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.10.7                                |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.10                               |
| Depends On    | TASK-M5.10.6                                |

## Description

Integrate the `CognitiveEngine` into the orchestrating `MessageProcessorService` to delegate incoming message turns to the active reasoning loop and remove the old passive LTM retrieval code from the request lifecycle.

## Work Items

1. **RED Phase**:
   - Modify unit tests in `internal/agent/application/service/message_processor_test.go` to verify that `ProcessIncomingMessage` leverages `CognitiveEngine`'s active reasoning loop instead of doing passive LTM lookups.
   - Assert that the passive RAG blocks are no longer injected prior to the loop.
   - Verify expected test compilation and execution failures (RED).

2. **GREEN Phase**:
   - Modify `internal/agent/application/service/message_processor.go` to inject and invoke `CognitiveEngine`.
   - Update `NewMessageProcessorService` constructor to accept `CognitiveEngine`.
   - Delete the old manual passive `retrieveLTMContext` call from the request path.
   - Ensure the user and assistant turns are still appended to Redis STM and eviction consolidation is triggered in the background.
   - Verify that all unit and integration tests compile and run green (GREEN).

3. **REFACTOR Phase**:
   - Clean up any unused parameters, helpers, and fields in `MessageProcessorService`.

## Acceptance Criteria

1. The orchestrator `MessageProcessorService` correctly delegates reasoning execution to `CognitiveEngine`.
2. Old passive LTM searches are completely removed from the request ingress lifecycle.
3. Redis STM conversation turns appending and background eviction consolidation are preserved intact.
