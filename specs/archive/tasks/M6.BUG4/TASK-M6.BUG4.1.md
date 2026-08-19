# TASK-M6.BUG4.1: Verify and Reconcile Agent Deployment Status in AgentService

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.BUG4.1                              |
| Status        | OPEN                                        |
| Spec          | BUG-M6.4                                    |
| Depends On    | none                                        |

## Description

The `AgentService` in `internal/keeper/application/service/agent_service.go` currently returns a hard error when trying to assign an already assigned agent to the same community, or when trying to unassign an already unassigned agent.
We must modify `Assign()` and `Unassign()` to verify if the agent is actually deployed and running (or undeployed and stopped) via the `CRDCoordinator.GetAgentCRDStatus` port, and reconcile/correct the deployment state accordingly.

### Scope

This task covers a single logical boundary: **Keeper Application Service** (`agent_service.go`) and its unit tests (`agent_service_test.go`).

## Work Items

1. **RED Phase**:
   - In `internal/keeper/application/service/agent_service_test.go`, add unit tests:
     - `TestAgentService_Assign_AlreadyAssigned_Running` (asserts success, mock `GetAgentCRDStatus` returning PhaseRunning, no CRD submission is triggered).
     - `TestAgentService_Assign_AlreadyAssigned_NotRunning` (asserts success, mock `GetAgentCRDStatus` returning nil/not-running phase, asserts async CRD submission is triggered, status updated to Pending).
     - `TestAgentService_Unassign_AlreadyUnassigned_NotRunning` (asserts success, mock `GetAgentCRDStatus` returning nil/terminated, asserts no CRD teardown is triggered).
     - `TestAgentService_Unassign_AlreadyUnassigned_Running` (asserts success, mock `GetAgentCRDStatus` returning PhaseRunning, asserts async CRD teardown is triggered).
   - Run tests and verify they fail (RED).

2. **GREEN Phase**:
   - Implement reconciliation logic in `AgentService.Assign`:
     - If agent's `CommunityID` is already assigned to the target `communityID`:
       - Query `s.crdCoordinator.GetAgentCRDStatus`.
       - If phase is `Running` or `Idle`, return successfully.
       - Otherwise, update agent status in database to `AgentStatusPending` and trigger async `SubmitAgentCRD`.
   - Implement reconciliation logic in `AgentService.Unassign`:
     - If agent's `CommunityID` is nil:
       - Query `s.crdCoordinator.GetAgentCRDStatus`.
       - If status is not nil and phase is not `Terminated`, trigger async `TeardownAgentCRD`, delete registration, invalidate cache keys, and publish offline status.
       - Return successfully.
     - If agent's `CommunityID` is not nil but doesn't match the target `communityID`, return an error.
   - Run tests and verify they pass (GREEN).

3. **REFACTOR Phase**:
   - Clean up code, check error handling, verify compliance with domain/application boundaries.
   - Ensure context propagation and logger structured fields are intact.

## Acceptance Criteria

1. Redundant `Assign` calls to the same community:
   - Check deployment status.
   - If running/idle, return `nil` without redeploying.
   - If not running, update database status to pending and asynchronously call `SubmitAgentCRD`.
2. Redundant `Unassign` calls:
   - Check deployment status.
   - If CRD exists and is not terminated, asynchronously call `TeardownAgentCRD`, delete registration, invalidate caches, publish offline status, and return `nil`.
   - If CRD is terminated or missing, return `nil` directly.
3. Unit tests cover all combinations of redundant assign/unassign and CRD status.
