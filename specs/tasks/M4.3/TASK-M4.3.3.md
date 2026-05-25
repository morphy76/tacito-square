# TASK-M4.3.3: Reconciler Metrics & Readiness Probes Setup

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.3.3                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.3                                |
| Depends On    | TASK-M4.3.1                                 |

## Description

Design and implement the Prometheus custom metrics collectors inside the reconciler context and establish a parallel Kubernetes client health check hook within the Operator's `/readyz` HTTP probe. The probe must ping the Kubernetes API server using a configurable context deadline, returning `503 Service Unavailable` on loss of connection, complying with K8s best practices.

## Boundary & Target Functions

- **Package**: `internal/operator`
- **File**: `internal/operator/bootstrap.go`
- **Target Functions**:
  - `NewServer()` (enhanced to support checker dependencies)
  - `(r *TacitoAgentReconciler) SetupWithManager(mgr ctrl.Manager)` (to wire up custom metrics)

## Work Items

1. **RED Phase**:
   * Implement unit tests verifying:
     * `/readyz` returns `200 OK` when the Kube client connectivity checker returns no errors.
     * `/readyz` returns `503 Service Unavailable` when Kube API connectivity returns a network error.
     * Exporter `/metrics` serves standard metric descriptions for `tacito_operator_reconciliation_total`, `tacito_operator_reconciliation_duration_seconds`, and `tacito_operator_active_agents`.

2. **GREEN Phase**:
   * Register custom Prometheus collectors (`prometheus.MustRegister`) inside bootstrap logic.
   * Inside the `Reconcile` loop, record step latencies to the histogram and increment success/error counters.
   * Implement a custom readiness checker verifying the Kube-API client can communicate with the server (e.g. executing a lightweight namespace or server version ping).
   * Integrate this checker inside the `health.NewProbe` stack inside `bootstrap.go`.

3. **REFACTOR Phase**:
   * Minimize log verbose outputs on health probes to avoid filling up disk partitions.

## Acceptance Criteria

1. Health probe unit tests pass under simulated cluster connection failures.
2. Custom metrics expose valid Prometheus data layouts on HTTP scrapes.
3. API connections use strict propagation contexts with explicit deadlines.
