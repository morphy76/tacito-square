# BUG-M6.4: Redundant Assignment or Unassignment Fails to Reconcile Deployment Status

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M6.4                                                           |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M6 — Communities & Messaging                                       |
| Affects       | `internal/keeper/application/service/agent_service.go`             |
| Violates      | SPEC-FR-M3.7, SPEC-FR-M4.7                                         |
| Discovered    | Reported during deployment/reconciliation logic review in Milestone 6. |

## Problem Statement

Currently, if an operator or system logic calls `Assign` on an agent that is already assigned to the same community, the service returns a hard error: `"agent already assigned to community: <id>"`.
Similarly, calling `Unassign` on an agent already unassigned from a community returns a hard error: `"agent is not assigned to community: <id>"`.

Instead of raising static errors, these calls should verify if the agent is actually deployed and running in Kubernetes. If they are not in the desired state, they should trigger the appropriate asynchronous reconciliation task (redeploying the agent or tearing it down).

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| AgentService | [agent_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go) | Hard-fails on redundant assign/unassign. Needs to call `GetAgentCRDStatus` and trigger reconciliation. |
| AgentService Tests | [agent_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service_test.go) | Missing tests verifying reconciliation on redundant assignments or unassignments. |

## Impact

1. Redundant API calls (e.g. from state synchronization loops or retry mechanisms) trigger unnecessary errors instead of self-healing / reconciling state.
2. Inconsistent state between the database and the K8s cluster cannot be easily repaired by repeating/retrying the assignment/unassignment operation.

## Expected Behaviour

1. **Assigning an already assigned agent (to the same community)**:
   - The system MUST check the current deployment status of the agent via the CRD coordinator.
   - If the agent is already deployed and running (phase is `Running` or `Idle`), the system MUST return successfully without action.
   - If the agent is NOT deployed and running, the system MUST update its status in the database to `Pending` and trigger the CRD submission asynchronously to reconcile/redeploy it.
2. **Unassigning an already unassigned agent**:
   - The system MUST check the current deployment status of the agent via the CRD coordinator.
   - If the agent is still deployed or running (phase is not terminated/stopped/missing), the system MUST trigger the teardown CRD asynchronously, delete registration, invalidate cache keys, and publish NATS status updates.
   - If the agent is not deployed, the system MUST return successfully without action.

## Acceptance Criteria

1. Redundant `Assign` calls to the same community return successfully and reconcile the deployment if the CRD is missing or not running.
2. Redundant `Unassign` calls return successfully and reconcile teardown if the CRD is still present/running.
3. Unit tests verify the reconciliation behavior for both `Assign` and `Unassign` flows under various CRD states.
