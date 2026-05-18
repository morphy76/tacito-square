# SPEC-NFR-HEALTH: Dependency-Aware Health Probes

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-HEALTH                    |
| Status        | VERIFIED                           |
| Component     | keeper, agent, bff                 |

## Specification

1. Every component MUST expose `/healthz` (liveness) and `/readyz` (readiness).
2. `/healthz` MUST return 200 if the process is alive. No dependency checks. Logs are emitted just for failures (no noisy logging) and first success after a failure.
3. `/readyz` MUST check ALL architectural dependencies in parallel with a configurable timeout. Logs are emitted just for failures (no noisy logging) and first success after a failure.
4. If any dependency is unhealthy, `/readyz` MUST return 503 with per-dependency status.
5. Response format MUST be JSON, including dependency statuses and errors (if any).

### Per-Component Dependency Checks

| Component | Dependencies Checked |
|-----------|---------------------|
| Keeper | PostgreSQL ping, NATS connection, Redis ping, Cache Redis ping |
| Agent | NATS connection, Redis ping, Cache Redis ping, Qdrant ping |

## Acceptance Criteria

1. LivezHandler always returns 200 with meaningful response
2. ReadyzHandler returns 200 when all checkers pass with meaningful response
3. ReadyzHandler returns 503 when any checker fails, with detail per checker and meaningful response
4. Do not check HTTP endpoints, unless infrastructural (e.g. Redis, NATS).
5. PingChecker wraps `func(ctx) error` (e.g., `db.Ping`)
