# TASK-M3.8.1: Shared Observability Database Pool Prometheus Metrics Collector

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | none                                        |

## Description

Design and implement the Prometheus custom collector for the pgx connection pool (`pgxpool.Pool`) inside the shared observability library. The collector must capture active metrics (acquired, idle, total, and max connections) dynamically on scrapes, ensuring robust handling and zero panic states when the database pool configuration is nil at startup.

## Boundary & Target Functions

- **Package**: `internal/shared/observability`
- **File**: `internal/shared/observability/metrics.go`
- **Target Functions**:
  - `NewDBPoolCollector(pool *pgxpool.Pool) prometheus.Collector`
  - `RegisterDBPoolStats(pool *pgxpool.Pool)`

## Work Items

1. **RED Phase**:
   - Write comprehensive unit tests in `internal/shared/observability/metrics_test.go` to assert:
     - `RegisterDBPoolStats(nil)` executes safely without panicking.
     - `NewDBPoolCollector(nil)`'s `Describe` and `Collect` methods function cleanly and yield no panic when given a nil pool pointer.
   - Assert standard registration behavior of Gauges carrying names `db_pool_acquired_connections`, `db_pool_idle_connections`, `db_pool_total_connections`, and `db_pool_max_connections`.

2. **GREEN Phase**:
   - Declare the `dbPoolCollector` custom collector struct implementing `prometheus.Collector` interface (`Describe(chan<- *prometheus.Desc)` and `Collect(chan<- prometheus.Metric)`).
   - In `Collect`, extract the pool status via `pool.Stat()` and map `AcquiredConns()`, `IdleConns()`, `TotalConns()`, and `MaxConns()` to constant gauge metrics.
   - Ensure a nil guard checks `pool == nil` at the beginning of the collector logic to guarantee thread-safe, panic-free operations.
   - Implement `RegisterDBPoolStats` using `prometheus.MustRegister`.

3. **REFACTOR Phase**:
   - Optimize constant descriptions creation by caching them on instantiation of the custom collector.
   - Verify compliance with `SPEC-NFR-OBSERVABILITY` naming schemas and package dependency rules.

## Acceptance Criteria

1. Standard unit tests in `internal/shared/observability/` compile and pass cleanly.
2. The custom database pool collector registers gauges correctly and does not panic on nil/zero pool contexts.
3. No external infrastructure adapters are imported into the domain layer.
