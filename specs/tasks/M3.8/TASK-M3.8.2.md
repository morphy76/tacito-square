# TASK-M3.8.2: Keeper Bootstrapper Database Pool Metrics Wiring

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | TASK-M3.8.1                                 |

## Description

Integrate and wire the database pool metrics collector inside the Keeper orchestrator bootstrapper. This registers the PostgreSQL pool statistics on startup, ensuring that the `/metrics` Prometheus exposition endpoint dynamically collects and outputs the pool statistics along with application runtime metrics.

## Boundary & Target Functions

- **Package**: `internal/keeper`
- **File**: `internal/keeper/bootstrap.go`
- **Target Functions**:
  - `NewServer(pool *pgxpool.Pool) *gin.Engine`

## Work Items

1. **RED Phase**:
   - Create unit assertions in `internal/keeper/bootstrap_test.go` or a mock metrics check to verify:
     - The GET `/metrics` endpoint is registered and functional when the server is built with a nil pool.
     - Scraping the endpoint returns a Prometheus-formatted document without hanging.

2. **GREEN Phase**:
   - Modify `internal/keeper/bootstrap.go` to import the shared observability metrics capability.
   - Inside `NewServer(pool)`, conditionally invoke `observability.RegisterDBPoolStats(pool)` if a non-nil `pool` is supplied at boot time.
   - Verify that all endpoints run without failure under standard server initialization configurations.

3. **REFACTOR Phase**:
   - Clean up imports in `bootstrap.go` to keep unused packages eliminated.
   - Enforce that `/metrics` route configuration remains fully excluded from global middleware request logging to prevent log pollution.

## Acceptance Criteria

1. Keeper standard tests compile and pass cleanly without panic.
2. The `/metrics` endpoint returns standard Prometheus text payload containing database pool status metrics when the pool is registered.
3. No GORM database driver implementation details leak outside the adapter boundaries.
