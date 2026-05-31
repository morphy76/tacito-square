# TASK-M5.3.4: Bootstrap Wiring, Health Probes, and End-to-End Verification

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M5.3.4                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M5.3                                |
| Depends On    | TASK-M5.3.3                                 |

## Description

Perform final bootstrap wiring for Redis connectivity inside the Agent startup sequence. Update the parallel `/readyz` health probe to check Redis availability as mandated by `RULE[k8s-best-practices.md]`. Align configuration keys with Viper bindings and verify E2E request processing with Redis memory.

## Work Items

1. **RED Phase**:
   - Write or update bootstrap integration tests (e.g. `internal/agent/bootstrap_test.go`) and health check tests to assert that `/readyz` fails when Redis is down or unavailable (but `/healthz` remains healthy).
   - Verify tests fail or reflect incomplete check signatures (RED).

2. **GREEN Phase**:
   - Update `internal/agent/bootstrap.go` to initialize the Redis connection pool using Viper configurations:
     - `TS_AGENT_REDIS_URL` (Redis server address).
     - `TS_AGENT_STM_TTL` (TTL duration, default `24h`).
     - Wire `RedisMemoryAdapter` into the `MessageProcessorService` constructor.
   - Update the `/readyz` HTTP endpoint handler to perform a parallel `Ping` check on the Redis client as part of the readiness dependency check.
   - Run tests and verify the complete bootstrap wiring and health probes pass successfully (GREEN).

3. **REFACTOR Phase**:
   - Clean up environment variables configuration mapping.
   - Ensure the Redis client pool limits and timeouts comply with `RULE[cloud-first.md]` (Bulkheads and timeouts).

## Acceptance Criteria

1. Agent bootstrap wires all dependencies and successfully connects to Redis.
2. The `/readyz` probe checks Redis connectivity in parallel and reports `503 Service Unavailable` with per-dependency details if Redis is unreachable.
3. The overall conversation flow with memory stores and retrieves history entries correctly.
