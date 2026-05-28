# BUG-M4.1: Assigned Agent Pods Fail to Deploy Due to Stubbed Operator Reconciliation Service

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M4.1                                                           |
| Status        | VERIFIED                                                           |
| Severity      | HIGH                                                               |
| Milestone     | M4 — Operator Core                                                 |
| Affects       | TASK-M4.3.1, TASK-M4.3.2, TASK-M4.3.3                              |
| Violates      | SPEC-FR-M4.3 (Reconciliation Controller)                           |
| Discovered    | Integration testing of agent-community assignment                  |

## Problem Statement

When a user assigns a `TacitoAgent` to a `TacitoCommunity`, the corresponding Kubernetes Pod is not deployed in the cluster, failing to satisfy the core deliverables of Milestones 3 and 4.

Two underlying issues contribute to this defect:
1. **Reconciliation Controller is a Stub in Operator**:
   - The operator's reconciler application service (`internal/operator/application/service/reconcile_service.go`) implements the `Reconcile` method, but it is currently just a **stub** that logs a message and returns `nil`. It does not construct or apply the actual Kubernetes Deployment or Service resources to the cluster.
   - Consequently, when the Keeper successfully creates and submits a `TacitoAgent` Custom Resource to the Kubernetes API server, the Operator does not actually deploy the corresponding agent Pod or headless Service.
2. **Inconsistent Assignment vs. Deployment Lifecycle**:
   - `SPEC-FR-M3.7` (Agent-Community Assignment) specifies that assigning an Agent to a Community automatically triggers CRD creation and pod deployment.
   - `SPEC-FR-M4.7` (Agent & Community Lifecycle Management REST API) introduces explicit `/deploy` and `/undeploy` endpoints and a `stopped` state:
     - `stopped`: *"Persisted database status indicating the record exists and is assigned to a community, but no active `TacitoAgent` CRD or Pod is provisioned in the cluster."*
   - Currently, the `Assign` method in `internal/keeper/application/service/agent_service.go` still automatically triggers `crdCoordinator.SubmitAgentCRD` in the background, bypassing the `stopped` state and conflicting with the M4 decoupled lifecycle model.
   - Furthermore, assigning an agent updates its status in the database to `stopped` (via `AssignToCommunity` in `agent_repository.go`), which directly contradicts immediate background deployment.

## Affected Files and Packages

- `internal/operator/application/service/reconcile_service.go` (Stubbed `Reconcile` implementation)
- `internal/keeper/application/service/agent_service.go` (Incorrectly couples assignment to immediate background CRD submission without appropriate state transitions)
- `specs/functional/M3/SPEC-FR-M3.7.md` (Needs sync with the M4 decoupled lifecycle)
- `specs/functional/M4/SPEC-FR-M4.7.md` (Needs sync with the automatic deployment upon assignment behavior)

## Impact

1. **No runtime containers are created**: Assigning an agent to a community leaves the resource as an un-reconciled CRD in Kubernetes with zero backing pods.
2. **State Mismatch**: The database state remains `stopped` even though CRD creation has been triggered asynchronously, resulting in an out-of-sync system state.
3. **Milestone Failure**: The operator fails to reconcile `TacitoAgent` custom resources into actual cluster workloads.

## Expected Behaviour

1. **Operator Reconciler**:
   - The Operator's `Reconcile` method in `ReconcileAgentServiceImpl` must be fully implemented to build the Deployment and Service manifests using `BuildDeployment` and `BuildHeadlessService`, apply them to the Kubernetes API server using the controller-runtime client, and update the custom resource's status phase (`Pending` / `Running` / `Error`).
2. **Unified Assignment & Deployment Lifecycle**:
   - **Assignment**: Stamping community association automatically submits the `TacitoAgent` CRD to deploy the runtime, transitioning the Agent status from `defined` to `pending`.
   - **Unassignment**: Deleting community association automatically deletes the `TacitoAgent` CRD, transitioning the Agent status from `assigned`/`stopped`/`running` to `defined`.
   - **Manual Lifecycle Controls**:
     - `/deploy` and `/undeploy` endpoints from `SPEC-FR-M4.7` allow manual overrides of this assigned state.
     - **Undeploy**: Suspends the running agent by deleting its `TacitoAgent` CRD and setting status to `stopped` without removing database community assignment.
     - **Deploy**: Resumes the stopped agent by recreating its `TacitoAgent` CRD and setting status back to `pending`.

## Acceptance Criteria

1. **Full Operator Reconciliation**:
   - Creating a `TacitoAgent` Custom Resource triggers the Operator to create a Deployment and a headless Service named after the custom resource.
   - The operator updates the Custom Resource status `status.phase` to `Pending` initially, and to `Running` once the pod becomes ready.
2. **Keeper Assignment Integration**:
   - Assigning an Agent to a Community via `POST /api/v1/communities/:id/agents/:id` updates the database community association, sets the Agent's status to `pending`, and triggers CRD submission.
   - Unassigning an Agent via `DELETE /api/v1/communities/:id/agents/:id` deletes the CRD and sets the Agent's status to `defined`.
3. **Manual Override Support**:
   - Calling `/undeploy` on an active agent deletes the CRD, sets status to `stopped`, and tears down the pod.
   - Calling `/deploy` on a stopped agent submits the CRD, sets status to `pending`, and spins up the pod.
4. **Validation Suite**:
   - All unit, integration, and reconciler tests pass without regression.

## Test Plan

### Automated Tests
- **Operator Reconciler Tests**: Verify that `Reconcile` creates/updates/deletes the Deployment and Service objects.
- **Keeper Service Tests**: Assert database status and CRD coordinator calls for the assignment, unassignment, deploy, and undeploy lifecycle sequences.
