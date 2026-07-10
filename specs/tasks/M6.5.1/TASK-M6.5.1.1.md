# TASK-M6.5.1.1: Keeper Domain Model — AgentRole & CommunityAssignment Aggregate

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.1.1 |
| Status      | DRAFT |
| Spec        | SPEC-FR-M6.5.1 |
| Depends On  | none |

## Description

Introduce the `AgentRole` typed enum and the `CommunityAssignment` aggregate into the keeper domain layer, and remove the `Role string` field (plus its validation) from the `Agent` aggregate. This task is strictly confined to `internal/keeper/domain/model/` — no application or adapter imports are permitted.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/domain/model/community_assignment_test.go`:
     - Test `CommunityAssignment.Validate()` passes for all valid role/topology combinations.
     - Test rejects zero-value `CommunityID`, `AgentID`, or `TenantID`.
     - Test rejects an unrecognized `AgentRole` value.
   - Update `internal/keeper/domain/model/agent_test.go`:
     - Remove any test case that supplies or validates a `Role` field on `Agent`.
     - Add a test asserting that `Agent.Validate()` no longer errors for any `Role` value.

2. **GREEN Phase**:
   - Create `internal/keeper/domain/model/community_assignment.go`:
     - Define `AgentRole` type with constants `AgentRoleStandalone`, `AgentRoleHub`, `AgentRoleSpoke`.
     - Define `CommunityAssignment` struct with fields: `CommunityID uuid.UUID`, `AgentID uuid.UUID`, `TenantID string`, `Role AgentRole`, `InformedAt *time.Time`, `AssignedAt time.Time`.
     - Implement `Validate() error` enforcing non-nil IDs, non-empty TenantID, and a valid `AgentRole` value.
   - Modify `internal/keeper/domain/model/agent.go`:
     - Remove the `Role string` field from the `Agent` struct.
     - Remove the `Role` validation block from `Agent.Validate()`.

3. **REFACTOR Phase**:
   - Ensure `AgentRole` constants are the single source of truth; remove any inline string literals for `"hub"`, `"spoke"` from the domain package.
   - Verify zero external imports in `community_assignment.go`.

## Acceptance Criteria

1. `community_assignment_test.go` unit tests all pass GREEN.
2. `agent_test.go` compiles and passes with `Role` field removed.
3. `community_assignment.go` imports only standard library and `github.com/google/uuid`.
4. `agent.go` contains no `Role` field or role-related validation.
