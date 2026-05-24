# TASK-M3.8.7: OpenTelemetry Database Client Query Tracing & Outbound Latency Metrics

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.8.7                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.8                                |
| Depends On    | TASK-M3.8.1                                 |

## Description

Design and implement distributed tracing and query metrics instrumentation for all outbound PostgreSQL interactions inside the Keeper orchestrator. Every SQL query must emit structured OpenTelemetry trace spans nested under the active REST context, and record outbound call latencies to the `outbound_dependency_duration_seconds` Prometheus histogram.

## Boundary & Target Functions

- **Packages**:
  - `internal/keeper/adapters/outbound/postgres`
  - `internal/shared/observability`
- **Files**:
  - `internal/keeper/adapters/outbound/postgres/agent_repository.go` (and all other repo adapters)
  - `internal/shared/observability/metrics.go`
- **Target Functions**:
  - Repository querying methods (`Create`, `GetByID`, `List`, `Update`, `Delete`)
  - Outbound latency tracking callbacks

## Work Items

1. **RED Phase**:
   - Write unit assertions in `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` or a mock tracing context:
     - Verify that calling repository methods under an active trace context generates a child span carrying standard attributes (`db.system = "postgresql"`, `db.statement`).
     - Verify that calling repository methods records query durations into the `outbound_dependency_duration_seconds` metric carrying `dependency="postgresql"` and operation tags.

2. **GREEN Phase**:
   - Instrument all repository database operations with OpenTelemetry `trace.Start` using the `db.query` span naming convention.
   - Inject the standard database semantic attributes into the trace span context:
     - `db.system = "postgresql"`
     - `db.statement = <sql_query>`
   - Wrap query execution blocks with standard Prometheus duration timers:
     ```go
     start := time.Now()
     // execute query...
     duration := time.Since(start).Seconds()
     observability.OutboundDependencyDuration.WithLabelValues("postgresql", operation, status).Observe(duration)
     ```
   - Ensure spans cleanly capture errors on exceptions before completing their lifecycle.

3. **REFACTOR Phase**:
   - Refactor repository query execution blocks to use clean, uniform wrappers to minimize instrumentation boilerplate code.
   - Verify that all context parameters are propagated safely across the entire query life cycle.

## Acceptance Criteria

1. Database queries emit correlated child spans nested under active HTTP handler spans.
2. The `/metrics` endpoint exposes query duration histograms (`outbound_dependency_duration_seconds`) with `dependency="postgresql"` labels.
3. No telemetry errors or metric collection loops block standard database repository execution.
