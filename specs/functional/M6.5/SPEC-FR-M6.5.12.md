# SPEC-FR-M6.5.12: STM Key Isolation for Non-Conversational Engagements

| Field       | Value |
|-------------|-------|
| ID          | SPEC-FR-M6.5.12 |
| Status      | DRAFT |
| Milestone   | M6.5 |
| Component   | agent |
| Depends On  | SPEC-FR-M5.3 |
| Supersedes  | none |

## Context

Short-term memory (STM, backed by Redis) maintains conversational context across turns within a thread. A conversational engagement is identified by a `thread_id` propagated in the NATS message. However, agents also serve **non-conversational** engagements — single-turn requests or short multi-step jobs without a persistent thread. Without explicit key isolation, parallel non-conversational requests to the same agent instance share or collide on Redis keys. This spec defines the Redis key schema for both modes and the TTL semantics of each.

## Specification

### 1. Redis Key Schema

**Conversational mode** (thread_id present in inbound message):
```
{key_namespace}:{tenant_id}:{community_id}:{agent_id}:thread:{thread_id}:{suffix}
```

**Non-conversational mode** (no thread_id in inbound message):
```
{key_namespace}:{tenant_id}:{community_id}:{agent_id}:session:{session_id}:{suffix}
```

Where:
- `key_namespace` = `Agent.ShortTermMemory.KeyNamespace` (defaults to agent name)
- `session_id` = a UUID v4 generated **once per inbound message** by the agent at the start of processing
- `suffix` = implementation-specific key discriminator (e.g., `history`, `state`)

### 2. Session ID Generation

For non-conversational mode:
1. At the start of `ExecuteReasoningLoop()`, check for the presence of `thread_id` in the incoming context.
2. If absent, generate a new `session_id = uuid.New()` and store it in the request-scoped `context.Context`.
3. The `session_id` is used for all STM reads/writes during that reasoning loop execution.
4. The `session_id` is **not returned** to the caller; it exists only for within-request key isolation.

### 3. TTL Semantics

| Mode | TTL Source | Default |
|------|------------|---------|
| Conversational (thread) | `ShortTermMemoryConfig.TTLSeconds` | 3600s |
| Non-conversational (session) | `ShortTermMemoryConfig.SessionTTLSeconds` (new field) | 300s |

Session keys have a short TTL because no caller is expected to reuse the session_id. Expired keys are cleaned up automatically by Redis.

### 4. STM Adapter Interface

The STM adapter must expose a key-prefix resolver:
```go
// ResolveKeyPrefix returns the Redis key prefix for the given context.
// Uses thread_id from ctx if present, otherwise uses a session_id from ctx.
func ResolveKeyPrefix(ctx context.Context, tenantID, communityID, agentID string) string
```

The cognitive engine injects either `thread_id` or a freshly-generated `session_id` into the context before calling any STM operations.

### 5. Configuration

Add `session_ttl_seconds int` to `ShortTermMemoryConfig` in the agent domain model. Default: 300. Configurable via Viper key `agent.stm.session_ttl_seconds`.

## Acceptance Criteria

1. Two concurrent non-conversational NATS messages to the same agent produce different `session_id` values and use different Redis key prefixes.
2. Two turns in the same thread (same `thread_id`) use the same Redis key prefix.
3. Session keys in Redis expire after `SessionTTLSeconds`; thread keys expire after `TTLSeconds`.
4. An inbound message with no `thread_id` always triggers session-mode key isolation.
5. Session TTL is configurable independently from thread TTL.

## Test Plan

- **Unit**: `ResolveKeyPrefix` returns thread-scoped key when `thread_id` is in context.
- **Unit**: `ResolveKeyPrefix` returns session-scoped key (with unique UUID) when `thread_id` is absent.
- **Unit**: Two calls with no `thread_id` produce different key prefixes.
- **Integration**: Two parallel non-conversational NATS messages do not overwrite each other's STM history.

## Files Affected

- `internal/agent/domain/model/memory.go` [MODIFY] — add `SessionTTLSeconds` to `ShortTermMemoryConfig`
- `internal/agent/adapters/outbound/redis/stm_adapter.go` [MODIFY] — `ResolveKeyPrefix`, TTL branching
- `internal/agent/application/service/cognitive_engine.go` [MODIFY] — session_id injection into context
- `internal/agent/application/service/cognitive_engine_test.go` [MODIFY] — add key isolation tests
