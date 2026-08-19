# TASK-M6.5-T3: Keeper Registry & Pruning Service

| Field       | Value                                 |
|-------------|---------------------------------------|
| Task ID     | TASK-M6.5-T3                          |
| Spec        | SPEC-FR-M6.5                          |
| Boundary    | Keeper Registry Ingestion & Pruning   |
| Status      | VERIFIED                              |
| Depends On  | TASK-M6.5-T2                          |

## Objective

Implement the Keeper wildcard NATS subscriber to receive and validate agent heartbeats, write them dynamically to PostgreSQL and the Redis cache, and build a background registry pruner task that transitions silent agents (missing heartbeats for >30s) to `offline`.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/inbound/nats/registry_subscriber.go` | NEW |
| `internal/keeper/adapters/inbound/nats/registry_subscriber_test.go` | NEW |
| `internal/keeper/application/service/registry_pruner.go` | NEW |
| `internal/keeper/application/service/registry_pruner_test.go` | NEW |
| `internal/keeper/bootstrap.go` | MODIFY |

## RED Phase

1. Create integration test `TestRegistry_IngestionAndPruning` in `internal/keeper/adapters/inbound/nats/registry_subscriber_test.go`.
2. Connect to NATS and Postgres test containers. Publish a heartbeat on `ts.community.test-comm.agent.test-agent.heartbeat`. Assert database agent record becomes `running` and `card` column is populated.
3. Simulate/mock 30s passing without a heartbeat and trigger the pruner; assert that database agent status changes to `offline` and its cache record is evicted.
4. Run tests and verify failure (RED).

## GREEN Phase

1. Create `internal/keeper/adapters/inbound/nats/registry_subscriber.go` subscribing to `ts.community.*.agent.*.heartbeat`.
2. Parse, validate, and extract trace headers from NATS messages.
3. On message:
   - Validate `AgentCard` JSON structure.
   - Update Postgres `agents` table: `status = 'running'`, `card = {payload}`, `updated_at = NOW()`.
   - Store the card in Redis cache mapping community ID to active cards.
4. Create `internal/keeper/application/service/registry_pruner.go` running a background loop.
5. In the pruner:
   - Identify agents with status `running` whose `updated_at` is older than 30s.
   - Update their database status to `offline`.
   - Evict from Redis cache and publish status change event to NATS: `ts.community.{community_id}.agent.{agent_id}.status` with status `offline`.
6. Wire subscriber and pruner in `internal/keeper/bootstrap.go`.
7. Verify all tests pass (GREEN).

## REFACTOR Phase

- Ensure pgx query trace spans cover DB updates cleanly.
- Verify context propagation is active during async DB updates.
- Check Redis connections are released properly during cache eviction.
