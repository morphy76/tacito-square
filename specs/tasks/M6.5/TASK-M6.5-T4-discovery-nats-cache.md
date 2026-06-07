# TASK-M6.5-T4: Registry Discovery (NATS Request/Reply & Client Cache)

| Field       | Value                                       |
|-------------|---------------------------------------------|
| Task ID     | TASK-M6.5-T4                                |
| Spec        | SPEC-FR-M6.5                                |
| Boundary    | Discovery Queries & Client-side Cache       |
| Status      | DRAFT                                       |
| Depends On  | TASK-M6.5-T3                                |

## Objective

Build the NATS Request/Reply query listener on Keeper to serve active card lists, and implement a thread-safe client-side cache on the Agent that queries Keeper over NATS and invalidates reactively when receiving status updates.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/inbound/nats/registry_handler.go` | NEW |
| `internal/keeper/adapters/inbound/nats/registry_handler_test.go` | NEW |
| `internal/agent/adapters/outbound/cache/client_cache.go` | NEW |
| `internal/agent/adapters/outbound/cache/client_cache_test.go` | NEW |

## RED Phase

1. Create a test `TestClientCache_RetrievesAndInvalidates` in `internal/agent/adapters/outbound/cache/client_cache_test.go`.
2. Mock NATS connection. Trigger cache resolution for a community member. Assert it sends a NATS request to `ts.community.{community_id}.registry.request`, receives cards, and stores them.
3. Publish a status event shifting an agent offline. Assert the local cache evicts the card.
4. Verify tests fail (RED).

## GREEN Phase

1. Create `internal/keeper/adapters/inbound/nats/registry_handler.go` subscribing to `ts.community.*.registry.request`.
2. On request, query Redis (or PostgreSQL fallback) to fetch all active cards for the community, and reply with the JSON array payload.
3. Create `internal/agent/adapters/outbound/cache/client_cache.go` implementing a thread-safe in-memory cache using `sync.RWMutex`.
4. Implement cache get method: on miss, issue a NATS Request to the registry subject with a timeout (e.g. 500ms) to load the community cards.
5. In the cache service, subscribe to NATS status and heartbeat wildcards `ts.community.{community_id}.agent.*` to invalidate local entries reactively upon receiving offline events or update them upon heartbeats.
6. Verify all tests pass (GREEN).

## REFACTOR Phase

- Ensure client request deadlines use standard `context.Context` propagation.
- Optimize locking to prevent deadlock scenarios in high-concurrency loops.
