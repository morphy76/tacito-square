# BUG-M3.7: Health Probes Missing NATS and Redis Dependency Checks

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.7                                                           |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/bootstrap.go, internal/shared/health/health.go     |
| Violates      | SPEC-NFR-HEALTH §3, §13, §17                                       |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

According to the dependency-aware health check specifications defined in `SPEC-NFR-HEALTH`, every component must expose liveness (`/healthz`) and readiness (`/readyz`) endpoints, with `/readyz` checking all architectural dependencies in parallel.

For the **Keeper** component, the required dependencies to be evaluated in `/readyz` are:
*   PostgreSQL ping
*   NATS connection
*   Redis ping
*   Cache Redis ping

In the current M3 candidate implementation:
1. **Missing Dependency Checks**: The readiness check in `internal/keeper/bootstrap.go` only registers the PostgreSQL connection checker:
   ```go
   var checkers []health.Checker
   if pool != nil {
       checkers = append(checkers, health.PingChecker("postgres", pool.Ping))
   }
   ```
   It completely omits NATS, Redis, and Cache Redis checks.
2. **Missing Reduced-Noise Logging**: The spec dictates that "Logs are emitted just for failures (no noisy logging) and first success after a failure." However, the health probe library in `internal/shared/health/health.go` does not perform any logging at all.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `bootstrap.go` | `internal/keeper/bootstrap.go` | Omits checks for NATS, Redis, and Cache Redis in `/readyz`. |
| `health.go` | `internal/shared/health/health.go` | Missing reduced-noise logging for probe transitions (healthy -> unhealthy and first success after failure). |

## Impact

1. **Undetected Network Partitions**: If Keeper loses connection to NATS or Redis, the `/readyz` probe will continue to report `200 OK` (assuming Postgres is fine), preventing Kubernetes from dynamically removing the unhealthy pod from the service rotation.
2. **No Operational Logging**: Production operations teams will lack structured logs tracking exactly when a dependency transitioned to an unhealthy state, complicating diagnosis.
3. **Compliance Failure**: Violates SPEC-NFR-HEALTH requirements.

## Expected Behaviour

1. The `/readyz` handler for Keeper MUST evaluate PostgreSQL, Redis, Cache Redis, and NATS connectivity in parallel.
2. The health probe library must support a transition-logging mechanism: log an error only when a check fails, and log an informational message on the first successful check after a failure.

## Acceptance Criteria

1. `/readyz` returns 503 if any of PostgreSQL, Redis, Cache Redis, or NATS is unavailable.
2. The `/readyz` response lists the status of all four checkers in the JSON response payload.
3. Failed checks are logged, and recovery is logged exactly once (first success). Normal checks run silently (no noisy logging).
