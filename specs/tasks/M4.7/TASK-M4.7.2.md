# TASK-M4.7.2: Outbound CRD Coordinator Status Query

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.7.2                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M4.7                                |
| Depends On    | TASK-M4.7.1                                 |

## Description

Extend the outbound `CRDCoordinator` port interface and its concrete adapter implementation `K8sCRDCoordinator` to support real-time status queries of deployed `TacitoAgent` Custom Resources in the K8s cluster. The status query must interact with the Kubernetes API server using `sigs.k8s.io/controller-runtime/pkg/client` to fetch the observed status subresource.

## Boundary & Target Functions

- **Packages**: `internal/keeper/application/ports/outbound`, `internal/keeper/adapters/outbound/crd`
- **Files**:
  - `internal/keeper/application/ports/outbound/crd_coordinator.go`
  - `internal/keeper/adapters/outbound/crd/crd_coordinator.go`
- **Target Interfaces, Structs & Functions**:
  - `CRDCoordinator` (interface)
  - `K8sCRDCoordinator` (struct)
  - `(c *K8sCRDCoordinator) GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error)`

## Work Items

1. **RED Phase**:
   * Add a new method signature `GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error)` to the `CRDCoordinator` interface.
   * Add unit tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` verifying:
     * Querying an existing deployed `TacitoAgent` resource returns its correct `TacitoAgentStatus` (Phase, Replicas, Conditions).
     * Querying a non-existent agent returns a specific `NotFound` or `stopped` condition gracefully without raising a terminal connection error.
     * Context deadlines and timeouts are propagated down to the K8s controller client.

2. **GREEN Phase**:
   * Implement `GetAgentCRDStatus` in `internal/keeper/adapters/outbound/crd/crd_coordinator.go`:
     * Execute a `Get` request using controller-runtime client.
     * Return the populated `Status` subresource.
     * Map `apierrors.IsNotFound` to a returned nil status or a custom stopped-status representation.

3. **REFACTOR Phase**:
   * Verify all schema types are fully mapped without leakage of core `client-go` internals to the application layer.

## Acceptance Criteria

1. Verification tests pass successfully using mock Kube client clientsets.
2. Port implementations remain completely isolated within `adapters/outbound/crd/`.
3. The coordinate port returns well-defined, uncoupled custom resource status types.
