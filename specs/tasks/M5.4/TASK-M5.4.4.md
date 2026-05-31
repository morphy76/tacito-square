# TASK-M5.4.4: Inbound Reasoning Pipeline Integration, Retrieval-Augmented Generation, and Memory Consolidation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.4.4                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M5.4                                |
| Depends On    | TASK-M5.4.2, TASK-M5.4.3                    |

## Description

Orchestrate the complete LTM retrieval and async memory consolidation pipeline within `MessageProcessorService`. On message ingestion, perform active semantic search (RAG) and inject retrieved context blocks inside the reasoning prompt. Implement the passive memory consolidation pipeline: when the Redis active history sliding window (`TS_AGENT_STM_LIMIT`) is exceeded, pop older turns, generate compressed summaries asynchronously, embed them, and upsert them to Qdrant as `EntryTypeConversation` vectors. Ensure graceful fallback logic is active on Qdrant/Embedder outages.

## Work Items

1. **RED Phase**:
   - Update `internal/agent/application/service/message_processor_test.go` to assert:
     - Prompts sent to the LLM contain the retrieved semantic `<long_term_memory>` context block.
     - Writing turns beyond the active sliding window limits successfully pops the oldest turns and triggers the background memory consolidation worker.
     - Graceful degradation: Simulating Qdrant or Embedder downtime does not cause the request to fail, correctly logging a Zerolog warning and falling back to Redis STM context.
   - Run tests and witness failure (RED).

2. **GREEN Phase**:
   - Update `MessageProcessorService` inside `internal/agent/application/service/message_processor.go`:
     - Accept `LongTermMemory` and `Embedder` outbound ports in the constructor.
     - Active Retrieval: Generate query embedding -> Search Qdrant LTM (with threshold constraints) -> Format system prompt prefix -> Call LLM reasoning.
     - Passive Consolidation: When pops occur during the active Redis STM cycle, run the async consolidation helper (or spawn a lightweight goroutine with context safeguards) that summarizes the popped turns, generates a vector embedding, and calls `LTM.Save`.
     - Implement fallback wrappers to handle Qdrant and Embedder network failures gracefully without raising client-facing errors.
   - Run tests and verify they pass (GREEN).

3. **REFACTOR Phase**:
   - Clean up goroutine context propagation structures to prevent background memory leaks or premature context cancellations.
   - Ensure structured logging captures `trace_id` and `span_id` cleanly.

## Acceptance Criteria

1. Inbound processing pipeline successfully executes LTM similarity queries and injects semantic context into the reasoning prompt.
2. The asynchronous memory consolidation workflow triggers automatically on Redis STM turn limits, safely persisting evicted conversations in Qdrant.
3. Fallback boundaries are active: Qdrant or Embedder outages are caught, logged, and bypassed without disrupting core client requests.
