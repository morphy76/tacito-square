# SPEC-FR-M12.2: Shared Hive Short-Term Memory (Community STM)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M12.2                               |
| Status        | DRAFT                                       |
| Milestone     | M12                                         |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M12.1                               |
| Supersedes    | none                                        |

## Context

In standard communities, Short-Term Memory (STM) keys are isolated per agent. In a "Hive" community, all agents must share access to a single, unified thread history list. This spec outlines the keyspace shift and the concurrency locking mechanisms required to prevent history interleaving when multiple agents read/write to the same thread in parallel.

## Specification

### 1. Community-Scoped Memory Keyspace

1. **Keyspace Restructuring**:
   * Change the Redis keyspace format from agent-isolated to community-isolated:
     `ts:stm:{tenant_id}:{community_id}:{thread_id}`
2. **Actor Mapping**:
   * All memory entries must explicitly track the writing actor (Agent Name or User) inside a metadata block to reconstruct which agent uttered which response during reasoning turns.

### 2. Hive Concurrency Coordination & Locking

To prevent race conditions, double-appended user turns, and interleaved execution logs during concurrent reasoning steps:

1. **Distributed Write Locks**:
   * Any read, append, or clear operation on the community-scoped Redis keyspace MUST acquire a fine-grained distributed lock (e.g. via Redis/Redlock) keyed by the thread ID: `ts:lock:stm:{tenant_id}:{thread_id}`.
2. **Turn Sequencing & Buffering**:
   * If Agent A and Agent B run in parallel and attempt to write to the same thread history:
     * Writes are queued sequentially using NATS or Redis Streams.
     * The system serializes execution logs, prefixing turns with clear context boundaries to ensure the LLM receives a coherent conversation history.

## Acceptance Criteria

To be defined during Milestone 12 review.

## Test Plan

To be defined during Milestone 12 review.

## Files Affected

To be defined during Milestone 12 review.
