# SPEC-FR-M5.3: Short-Term Memory (Redis)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M5.3                                |
| Status        | DRAFT                                       |
| Milestone     | M5                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Agents maintain short-term conversation memory in Redis for fast access during active conversations. This is distinct from the infrastructure cache (SPEC-NFR-CACHE) and uses separate key namespacing.

## Specification

1. The system MUST define a `ShortTermMemory` outbound port in the agent domain layer.
2. The system MUST implement a Redis adapter for STM storage.
3. Keys MUST follow the prefix pattern `ts:agent:stm:{agent_id}:{thread_id}` to distinguish from cache keys.
4. Conversation turns MUST be stored as ordered entries with timestamps.
5. STM entries MUST have configurable TTL-based expiry (default: 24 hours).
6. All operations MUST be thread-safe.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
