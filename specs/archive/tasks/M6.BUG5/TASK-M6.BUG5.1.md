# TASK-M6.BUG5.1: Propagate Agent Role in CRD Coordinator

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.BUG5.1                              |
| Status        | VERIFIED                                    |
| Spec          | BUG-M6.5                                    |
| Depends On    | none                                        |

## Description

The `K8sCRDCoordinator` in `internal/keeper/adapters/outbound/crd/crd_coordinator.go` does not map the `Role` field of the `model.Agent` domain model to the `Role` field of `v1alpha1.TacitoAgentSpec` when submitting the custom resource. We need to update both the create and update paths inside `SubmitAgentCRD` to set `Spec.Role = agent.Role`.

### Scope

This task covers a single logical boundary: **Keeper CRD Coordinator Adapter** (`crd_coordinator.go`) and its unit tests (`crd_coordinator_test.go`).

## Work Items

1. **RED Phase**:
   - In `internal/keeper/adapters/outbound/crd/crd_coordinator_test.go`, add unit tests:
     - `TestSubmitAgentCRD_HubRolePropagated` (asserts that `agent.Role = "hub"` results in `fetched.Spec.Role == "hub"`).
     - `TestSubmitAgentCRD_SpokeRolePropagated` (asserts that `agent.Role = "spoke"` results in `fetched.Spec.Role == "spoke"`).
     - `TestSubmitAgentCRD_RoleUpdated` (asserts that when updating an existing CRD, `latest.Spec.Role` is successfully updated to match the new `agent.Role`).
   - Run the tests and verify that they fail (RED).

2. **GREEN Phase**:
   - Update `internal/keeper/adapters/outbound/crd/crd_coordinator.go`:
     - In the create path of `SubmitAgentCRD`, set `Role: agent.Role,` on the `v1alpha1.TacitoAgentSpec` struct.
     - In the update path (within the conflict-resolution retry loop), assign `latest.Spec.Role = agent.Role`.
   - Run the tests and verify they pass (GREEN).

3. **REFACTOR Phase**:
   - Clean up code, check formatting, and ensure complete test coverage.

## Acceptance Criteria

1. Submitting a new agent CRD propagates the agent's role ("hub" or "spoke") to the custom resource specification.
2. Updating an existing agent CRD successfully updates the custom resource specification's role to match the agent's new role.
3. Unit tests cover both role assignment on creation and role update on patch/edit.
