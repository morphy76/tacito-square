# SPEC-FR-04.2: Short-Term Memory (Redis Adapter)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-04.2                       |
| Status        | IMPLEMENTED                        |
| Milestone     | M2                                 |
| FR/NFR Ref    | FR-04.2                            |
| Component     | agent                              |
| Depends On    | SPEC-FR-01.2                       |

## Context

Agents need per-thread short-term memory (recent conversation context) backed by Redis with automatic TTL expiry.

## Specification

1. The `ShortTermMemory` outbound port MUST define:
   - `Store(ctx, entry MemoryEntry) error`
   - `Retrieve(ctx, agentID, threadID string, limit int) ([]MemoryEntry, error)`
2. Redis keys MUST be namespaced: `ts:stm:{community}:{thread}:{agent}:{timestamp}`.
3. Entries MUST have a configurable TTL (default: 1 hour).
4. `Retrieve` MUST return entries in chronological order (oldest first).
5. Entries MUST be serialized as JSON values.

## Acceptance Criteria

1. `Store` writes entry to Redis with correct key pattern
2. `Retrieve` returns entries in chronological order
3. TTL is applied to stored entries
4. Entries for different threads are isolated
5. Limit parameter caps the number of returned entries

## Test Plan

- Unit tests with Redis mock (fixture-based)
- Integration tests with testcontainers Redis
- 6+ test cases

## API Contract

N/A (outbound port, no HTTP surface)

## Files Affected

- `internal/agent/ports/outbound/ports.go` (port interface — already exists)
- `internal/agent/adapters/outbound/redis/stm_adapter.go` (NEW)
- `internal/agent/adapters/outbound/redis/stm_adapter_test.go` (NEW)
