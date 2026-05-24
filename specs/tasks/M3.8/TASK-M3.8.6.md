# TASK-M3.8.6: Dependency-Aware Parallel Health Probes (Readyz & Liveness)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.6                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | none                                        |

## Description

Design and implement comprehensive liveness (`/healthz`) and readiness (`/readyz`) HTTP endpoints inside the Keeper orchestrator in compliance with `k8s-best-practices.md`. The readiness probe must concurrently check all downstream backing services (PostgreSQL, NATS, Redis, and Cache Redis) in parallel within a configurable timeout, returning structured JSON mapping status details per-dependency and failing with a `503 Service Unavailable` if any dependency is offline.

## Boundary & Target Functions

- **Package**: `internal/keeper`
- **Files**:
  - `internal/keeper/bootstrap.go` (Server wiring)
  - `internal/shared/health/probe.go` (Or shared health library context)
- **Target Functions**:
  - `NewServer(pool *pgxpool.Pool) *gin.Engine`
  - Custom health checkers registration and parallel executor loop

## Work Items

1. **RED Phase**:
   - Write unit and handler tests using `gin.TestMode` in `internal/keeper/bootstrap_test.go` or dedicated probe tests to verify:
     - `/healthz` always returns `200 OK` with a valid JSON response carrying a clean status message.
     - `/readyz` returns `200 OK` with JSON details when all backing connections (PostgreSQL, Redis, Cache Redis, NATS) are online.
     - `/readyz` returns `503 Service Unavailable` with details and errors per-dependency when one or more of the connections are unreachable or nil.

2. **GREEN Phase**:
   - Wire `/healthz` and `/readyz` routes directly in `bootstrap.go`.
   - Implement the parallel execution loop for readiness checks:
     - Execute each backing check in its own goroutine using a short, configurable deadline timeout (e.g. 2-5 seconds).
     - Coordinate results using wait groups or channels to guarantee non-blocking, asynchronous verification.
     - Map checks for:
       - **PostgreSQL**: pool connection status / ping
       - **NATS**: active subscription / client ping
       - **Redis**: primary cache ping
       - **Cache Redis**: session cache ping
     - Construct the JSON payload containing status details for each backend (e.g., `{"postgres": "OK", "nats": "unreachable"}`).
   - Ensure logs are emitted only on failure states and on the first recovery success to avoid log pollution.

3. **REFACTOR Phase**:
   - Consolidate common checker wrappers inside a clean shared `health` packages module.
   - Assert standard JSON structure matches the required `{ "postgres": "OK", "nats": "error: ..." }` mapping rules.

## Acceptance Criteria

1. Both health probes are registered deterministically at server boot.
2. The `/readyz` endpoint validates PostgreSQL, NATS, Redis, and Cache Redis concurrently in parallel.
3. Failures in backing systems return detailed `503 Service Unavailable` responses listing precise dependency errors in standard JSON.
