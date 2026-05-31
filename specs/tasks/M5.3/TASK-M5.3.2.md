# TASK-M5.3.2: Redis Short-Term Memory Infrastructure Adapter

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.3.2                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.3                                |
| Depends On    | TASK-M5.3.1                                 |

## Description

Implement the concrete Redis infrastructure adapter (`RedisMemoryAdapter`) in the adapters layer. It must implement the `ShortTermMemory` outbound port, guarantee strict multi-tenant isolation through prefixing, store entries in a Redis `LIST`, enforce TTL-based expiry, and integrate standard OpenTelemetry tracing.

## Work Items

1. **RED Phase**:
   - Write integration tests inside `internal/agent/adapters/outbound/redis/memory_adapter_test.go` using a mock Redis client or a local container that asserts:
     - Strict multi-tenant data separation (tenant A keys cannot leak into tenant B).
     - Storage of entries serialized as JSON string arrays in a Redis list.
     - TTL-based expiry and correct limits in sliding window retrieval.
     - Outbound OTel tracing span coverage.
   - Verify tests fail since the implementation does not exist (RED).

2. **GREEN Phase**:
   - Create `internal/agent/adapters/outbound/redis/memory_adapter.go`.
   - Implement the `RedisMemoryAdapter` using the approved `github.com/redis/go-redis` client library.
   - Enforce key prefix logic: `ts:tenant:{tenant_id}:agent:{agent_id}:stm:{thread_id}`.
   - Configure TTL-based operations using `Expire` or `EXPIRE` commands on `Append`.
   - Wire standard OpenTelemetry instrumentation in the Redis client commands.
   - Run the integration tests and verify they pass (GREEN).

3. **REFACTOR Phase**:
   - Review Redis client pooling, connection pooling, resource cleanups, and ensure Go context propagation is actively handled for all commands.

## Acceptance Criteria

1. Redis adapter complies with the `ShortTermMemory` interface contract.
2. Redis keys match the strict tenant prefix structure: `ts:tenant:{tenant_id}:agent:{agent_id}:stm:{thread_id}`.
3. Every `Append` operation refreshes/assigns the configurable TTL expiration bound.
4. Tracing spans are correctly generated and propagated on Redis interactions.
