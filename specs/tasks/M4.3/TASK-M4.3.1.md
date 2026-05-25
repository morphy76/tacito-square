# TASK-M4.3.1: Reconciler Inbound Adapter & Service Setup

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.3.1                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.3                                |
| Depends On    | none                                        |

## Description

Design and implement the Operator reconciler loop using `sigs.k8s.io/controller-runtime` package interfaces. Structure the reconciler strictly as a Ports & Adapters **Inbound Driving Adapter** under `adapters/inbound/reconciler.go` that watches `TacitoAgent` Custom Resources and invokes a concrete application service (`application/service/reconcile_service.go`) containing the orchestration use cases.

## Boundary & Target Functions

- **Package**: `internal/operator/adapters/inbound` & `internal/operator/application/service`
- **Files**:
  - `internal/operator/adapters/inbound/reconciler.go`
  - `internal/operator/application/service/reconcile_service.go`
- **Target Structs & Functions**:
  - `TacitoAgentReconciler` (type struct)
  - `(r *TacitoAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)`
  - `ReconcileAgentService` (type struct)

## Work Items

1. **RED Phase**:
   * Setup integration tests in `internal/operator/adapters/inbound/reconciler_test.go` utilizing `envtest` control plane:
     * Assert that creating a mock `TacitoAgent` triggers the `Reconcile` loop.
     * Verify the reconciler logs are registered with structured JSON outputs (`zerolog`).
     * Verify that transient/not-found resource events yield peaceful `nil` error states without panic.

2. **GREEN Phase**:
   * Implement `Reconcile` method inside `reconciler.go`:
     * Fetch `TacitoAgent` resource by namespace/name key.
     * Call `ReconcileAgentService` to construct the backing Deployment and headless Service objects.
     * Set `OwnerReference` on child objects to tie their lifecycles to the parent.
     * Setup status condition writers.

3. **REFACTOR Phase**:
   * Optimize database scheme registration bounds.
   * Verify adherence to strict hexagonal boundary rules.

## Acceptance Criteria

1. Envtest reconciler mock tests compile and execute cleanly.
2. The core reconciliation loop operates out-of-band and delegates business mappings to `application/service/`.
