# SPEC-NFR-HEALTH: Dependency-Aware Health Probes

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-HEALTH                    |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-09.5, FR-09.6, FR-09.7, FR-09.8, FR-09.9 |
| Component     | shared, keeper, agent, bff, operator |

## Specification

1. Every component MUST expose `/healthz` (liveness) and `/readyz` (readiness).
2. `/healthz` MUST return 200 if the process is alive. No dependency checks.
3. `/readyz` MUST check ALL architectural dependencies in parallel with a configurable timeout.
4. If any dependency is unhealthy, `/readyz` MUST return 503 with per-dependency status.
5. Response format: `{"status": "ready|not_ready", "checks": [{"name": "...", "status": "healthy|unhealthy", "error": "..."}]}`

### Per-Component Dependency Checks

| Component | Dependencies Checked |
|-----------|---------------------|
| Keeper | PostgreSQL ping, NATS connection |
| Agent | NATS connection, Redis ping, Qdrant ping, LLM endpoint reachability |
| BFF | Keeper API reachability |
| Operator | K8s API server connectivity |

## Acceptance Criteria

1. LivezHandler always returns 200 with `{"status": "alive"}`
2. ReadyzHandler returns 200 when all checkers pass
3. ReadyzHandler returns 503 when any checker fails, with detail per checker
4. ReadyzHandler with no checkers returns 200
5. HTTPChecker detects unreachable endpoints
6. PingChecker wraps `func(ctx) error` (e.g., `db.Ping`)

## Files

- `internal/shared/health/health.go` ✅ IMPLEMENTED
- `internal/shared/health/health_test.go` ✅ 7 tests passing
