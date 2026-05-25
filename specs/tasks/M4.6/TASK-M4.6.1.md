# TASK-M4.6.1: Keeper Outbound CRD Coordinator Client Setup

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.6.1                                 |
| Status        | PLANNED                                     |
| Spec          | SPEC-FR-M4.6                                |
| Depends On    | none                                        |

## Description

Design and implement the `K8sCRDCoordinator` adapter inside the Keeper infrastructure layer (`adapters/outbound/crd/crd_coordinator.go`). The coordinator must initialize a real Kubernetes typed client using standard in-cluster or kubeconfig credentials and satisfy the `outbound.CRDCoordinator` driving interface. It must wrap calls using context deadlines and client-go `RetryOnConflict` handler.

## Boundary & Target Functions

- **Package**: `internal/keeper/adapters/outbound/crd`
- **File**: `internal/keeper/adapters/outbound/crd/crd_coordinator.go`
- **Target Structs & Functions**:
  - `K8sCRDCoordinator` (type struct)
  - `NewK8sCRDCoordinator(config *rest.Config) (*K8sCRDCoordinator, error)`
  - `(c *K8sCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) error`
  - `(c *K8sCRDCoordinator) TeardownAgentCRD(ctx context.Context, agent *model.Agent) error`

## Work Items

1. **RED Phase**:
   * Implement unit tests in `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go` utilizing a fake client (`fake.NewSimpleClientset` or similar):
     * Verify `SubmitAgentCRD` attempts to create/update the `TacitoAgent` resource.
     * Verify standard conflict update states (`409 Conflict`) are captured and successfully resolved using standard retry routines.
     * Verify that context deadlines (e.g. 5 seconds) are passed and abort Kube-API calls when exceeded.

2. **GREEN Phase**:
   * Implement `K8sCRDCoordinator` using the dynamic/typed client interface.
   * Inside `SubmitAgentCRD`, check if the custom resource exists:
     * If not, create the resource.
     * If yes, fetch and update it using `util/retry.RetryOnConflict`.
   * Inside `TeardownAgentCRD`, delete the resource safely.

3. **REFACTOR Phase**:
   * Ensure that the Kube API operations use stripped, compiled schema models from the shared package (`pkg/kubernetes/apis/tacito/v1alpha1`).

## Acceptance Criteria

1. Verification tests pass successfully using mock Kube client clientsets.
2. Port implementations remain completely isolated within `adapters/outbound/crd/`.
3. Outbound calls handle network conflicts using RetryOnConflict.
