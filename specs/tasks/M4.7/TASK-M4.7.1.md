# TASK-M4.7.1: Domain Status & Database Updates

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.7.1                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M4.7                                |
| Depends On    | none                                        |

## Description

Design and implement the extensions to domain statuses and GORM/pgx relational validations for both Agents and Communities in the Keeper domain. This includes adding new states (`stopped`, `pending`, `running`, `error`, `inactive`) to domain models and updating the repository assignment/unassignment status transitions.

## Boundary & Target Functions

- **Packages**: `internal/keeper/domain/model`, `internal/keeper/adapters/outbound/postgres`
- **Files**:
  - `internal/keeper/domain/model/agent.go`
  - `internal/keeper/domain/model/community.go`
  - `internal/keeper/adapters/outbound/postgres/agent_repository.go`
- **Target Types & Functions**:
  - `AgentStatus` (type enum)
  - `CommunityStatus` (type enum)
  - `Agent.Validate() error`
  - `Community.Validate() error`
  - `AgentRepository.AssignToCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error`
  - `AgentRepository.UnassignFromCommunity(ctx context.Context, agentID uuid.UUID, communityID uuid.UUID) error`

## Work Items

1. **RED Phase**:
   * Implement unit tests in `internal/keeper/domain/model/agent_test.go` and `community_test.go` verifying:
     * `Validate` permits the new `stopped`, `pending`, `running`, and `error` statuses for Agents.
     * `Validate` permits the new `inactive` status for Communities.
   * Implement integration tests in `internal/keeper/adapters/outbound/postgres/agent_repository_test.go` verifying:
     * `AssignToCommunity` transitions the assigned agent status to `stopped` (instead of the obsolete `assigned`).
     * `UnassignFromCommunity` transitions the unassigned agent status back to `defined`.

2. **GREEN Phase**:
   * Add `AgentStatusStopped`, `AgentStatusPending`, `AgentStatusRunning`, and `AgentStatusError` to `internal/keeper/domain/model/agent.go`.
   * Update `Agent.Validate()` to support these new status boundaries.
   * Add `CommunityStatusInactive` to `internal/keeper/domain/model/community.go`.
   * Update `Community.Validate()` to support the `inactive` status.
   * In `internal/keeper/adapters/outbound/postgres/agent_repository.go`:
     * Update `AssignToCommunity` to use `model.AgentStatusStopped` for the transition state.

3. **REFACTOR Phase**:
   * Ensure status checks use strictly-typed model constants instead of inline raw strings.

## Acceptance Criteria

1. Model validation unit tests pass successfully with new status values.
2. Integration repository assignment tests confirm successful transition to `stopped`.
3. Strict compile-time safety is maintained for all enum types.
