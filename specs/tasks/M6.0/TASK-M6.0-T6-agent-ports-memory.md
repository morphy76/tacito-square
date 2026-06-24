# TASK-M6.0-T6: Agent Ports & Redis Memory Rollback

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T6                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Agent Memory Layer — `internal/agent/adapters/outbound/redis` |
| Status      | VERIFIED                                                |
| Depends On  | TASK-M6.0-T1                                            |

## Objective

Extend the short-term memory outbound port and Redis adapter to support rolling back the last written conversation turn (in case of LLM failures).

## Files

| File | Action |
|------|--------|
| `internal/agent/application/ports/outbound/memory.go` | MODIFY |
| `internal/agent/adapters/outbound/redis/memory_adapter.go` | MODIFY |
| `internal/agent/application/service/message_processor_test.go` | MODIFY |
| `internal/agent/adapters/outbound/redis/memory_adapter_test.go` | MODIFY |

## RED Phase

Update Redis memory adapter tests in `internal/agent/adapters/outbound/redis/memory_adapter_test.go` and mock references:

- `TestRedisMemoryAdapter_RollbackLast`: Populate STM with two entries (e.g. entryA, entryB); call `RollbackLast`; retrieve history and verify only entryA remains. Call `RollbackLast` again; verify list is empty. Call `RollbackLast` on empty list; assert it returns no error.
- Update `MockShortTermMemory` in `message_processor_test.go` to include `RollbackLast`, raising compilation error if interface signature is missing.

Run `make test` — compile/test failure (RED).

## GREEN Phase

1. Modify `internal/agent/application/ports/outbound/memory.go` to add `RollbackLast(ctx context.Context, tenantID, agentID, threadID string) error` to `ShortTermMemory` interface.
2. Modify `internal/agent/adapters/outbound/redis/memory_adapter.go` to implement `RollbackLast` using Redis client `RPop`. Check for `redis.Nil` to handle empty list gracefully.
3. Modify `internal/agent/application/service/message_processor_test.go` to add `RollbackLast` method stub in `MockShortTermMemory`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm Redis `RPop` doesn't throw errors when operating on empty keys/lists.
- Ensure tracing span is created for `redis.rollback_last` and duration metrics are recorded.
