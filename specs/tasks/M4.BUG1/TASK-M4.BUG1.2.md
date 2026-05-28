# TASK-M4.BUG1.2: Keeper Assignment Pending Status Transition

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.BUG1.2                              |
| Status        | VERIFIED                                    |
| Spec          | BUG-M4.1                                    |
| Depends On    | none                                        |

## Description

Align the Keeper's database status update inside `AssignToCommunity` (`internal/keeper/adapters/outbound/postgres/agent_repository.go`) to transition the Agent's status from `defined` to `pending` upon assignment (matching our synchronized lifecycle where assignment triggers immediate CRD submission). Update associated unit and integration tests to verify this transition.

## Work Items

1. **RED Phase**:
   - [x] Locate and modify the assignment integration tests in `internal/keeper/adapters/outbound/postgres/agent_assignment_test.go` and mock repository assertions in HTTP handlers to expect the status transition to `pending` instead of `stopped`.
   - [x] Run the test suite and verify that these assertions fail (RED).

2. **GREEN Phase**:
   - [x] Modify the database `AssignToCommunity` transaction block in `agent_repository.go` to set the status string to `string(model.AgentStatusPending)` instead of `string(model.AgentStatusStopped)`.
   - [x] Ensure the tests compile and go GREEN.

3. **REFACTOR Phase**:
   - [x] Clean up any unused test mocks or duplicate assertions.

## Acceptance Criteria

1. Assigning an Agent to a Community updates its database status to `pending`.
2. Existing unit/integration tests are aligned with the new lifecycle model and pass successfully.
