# SPEC-FR-15: Usage Quotas

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-15                         |
| Status        | DRAFT                              |
| Milestone     | M4 (domain), M5 (enforcement)      |
| Component     | keeper, operator                   |
| Depends On    | SPEC-FR-05.1, SPEC-FR-01.1        |

## Context

Communities and agents need usage quotas to prevent runaway resource consumption and enable fair multi-tenancy.

## Specification

### FR-15.1: Community Quotas
1. Each community MUST support configurable quotas:
   - `max_agents` — maximum concurrent agent instances
   - `max_threads` — maximum concurrent open threads
   - `max_messages_per_hour` — rate limit across all threads
   - `max_storage_bytes` — S3 payload storage cap
2. Quotas are stored in the `community.quotas` JSONB field.
3. Default quotas MUST be configurable via Helm values.

### FR-15.2: Agent Quotas
1. Each agent instance MAY have individual quotas:
   - `max_tokens_per_hour` — LLM token consumption rate limit
   - `max_tool_calls_per_hour` — MCP invocation rate limit
   - `max_memory_entries` — short-term memory entry cap
2. Agent quotas are resolved at spawn time from skill configuration or community defaults.

### FR-15.3: Quota Enforcement
1. Quota checks MUST occur at the **service layer** before executing the guarded operation.
2. Exceeded quotas MUST return HTTP 429 (Too Many Requests) with `Retry-After` header.
3. Quota violations MUST be logged with the violating principal and quota details.
4. A `QuotaChecker` port MUST be defined in the outbound ports, backed by Redis counters with TTL-based windows.

### FR-15.4: Quota Tracking & Reporting
1. Current quota usage MUST be queryable via `GET /api/v1/communities/{id}/quotas`.
2. Agent-level usage MUST be queryable via `GET /api/v1/agents/{id}/quotas`.
3. Quota usage MUST be included in Prometheus metrics (gauge per quota type).

## Acceptance Criteria

1. Community creation with quotas persisted correctly
2. Spawn rejected when `max_agents` exceeded (429)
3. Thread creation rejected when `max_threads` exceeded (429)
4. Token consumption tracked and enforced per agent
5. Quota usage queryable via API
6. Prometheus gauges exported for quota utilization

## Files Affected

- `internal/keeper/domain/quota.go` (NEW)
- `internal/keeper/ports/outbound/quota_checker.go` (NEW)
- `internal/keeper/adapters/outbound/redis/quota_adapter.go` (NEW)
- `internal/keeper/domain/community.go` (MODIFY — add quotas field)
