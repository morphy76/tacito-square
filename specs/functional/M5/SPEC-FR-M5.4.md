# SPEC-FR-M5.4: Long-Term Memory (Qdrant)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.4                                |
| Status        | DRAFT                                       |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Agents store long-term knowledge in Qdrant vector database for semantic retrieval across conversations. This enables agents to recall relevant past interactions and accumulated knowledge.

## Specification

1. The system MUST define a `LongTermMemory` outbound port in the agent domain layer.
2. The system MUST implement a Qdrant adapter for vector storage and retrieval.
3. Each agent MUST have its own Qdrant collection, named `ts_agent_{agent_id}`.
4. The adapter MUST support: store (embed + upsert), search (query by similarity), delete.
5. Embedding generation MUST use the configured LLM provider's embedding model.
6. Similarity search MUST return configurable top-K results with score thresholds.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
