# TASK-M6.5.1.2: Keeper Application Layer — Ports & Assignment Service

| Field       | Value |
|-------------|-------|
| ID          | TASK-M6.5.1.2 |
| Status      | IMPLEMENTED |
| Spec        | SPEC-FR-M6.5.1 |
| Depends On  | TASK-M6.5.1.1 |

## Description

Update the keeper application layer to reflect the new assignment contract. This covers three boundaries within `internal/keeper/application/`:
- **Inbound port** (`ports/inbound/usecases.go`): update `AssignmentUseCase` with `role model.AgentRole` parameter on `Assign` and a new `ListByCommunity` method.
- **Outbound port** (`ports/outbound/community_assignment_repository.go`): introduce the `CommunityAssignmentRepository` interface.
- **Service** (`service/community_service.go`): implement topology/role validation logic and wire the new outbound port.

No adapter or infrastructure imports are permitted in this layer.

## Work Items

1. **RED Phase**:
   - Create `internal/keeper/application/service/community_assignment_service_test.go`:
     - Mock `CommunityAssignmentRepository` and `CommunityRepository` using testify mocks.
     - Test `Assign` auto-sets `standalone` for `single-agent` topology ignoring caller role.
     - Test `Assign` accepts `hub` for `hub-spoke` topology when `CountHubs` returns 0.
     - Test `Assign` rejects a second `hub` for `hub-spoke` topology with a domain conflict error.
     - Test `Assign` rejects `hub` role for `single-agent` topology with a domain conflict error.
     - Test `Assign` rejects `standalone` role for `hub-spoke` topology with a domain conflict error.
     - Test `ListByCommunity` delegates to the repository and returns results.
     - Test `Unassign` delegates to the repository delete and updates agent status.

2. **GREEN Phase**:
   - Create `internal/keeper/application/ports/outbound/community_assignment_repository.go`:
     - Define `CommunityAssignmentRepository` interface with methods: `Create`, `Delete`, `ListByCommunity`, `CountHubs`, `CountByCommunity`.
   - Modify `internal/keeper/application/ports/inbound/usecases.go`:
     - Update `AssignmentUseCase.Assign` signature to `Assign(ctx context.Context, communityID uuid.UUID, agentID uuid.UUID, role model.AgentRole) error`.
     - Add `ListByCommunity(ctx context.Context, communityID uuid.UUID) ([]*model.CommunityAssignment, error)` to `AssignmentUseCase`.
   - Modify `internal/keeper/application/service/community_service.go`:
     - Inject `CommunityAssignmentRepository` and `CommunityRepository` (for topology lookup).
     - Implement `Assign`: load community topology, resolve/validate role, call `CommunityAssignmentRepository.Create`.
     - Implement `Unassign`: call `CommunityAssignmentRepository.Delete`.
     - Implement `ListByCommunity`: delegate to `CommunityAssignmentRepository.ListByCommunity`.

3. **REFACTOR Phase**:
   - Extract topology/role validation into a private `validateRole(topology CommunityTopology, role AgentRole) error` helper to keep `Assign` clean.
   - Ensure all service methods propagate `context.Context` and return wrapped errors.

## Acceptance Criteria

1. All unit tests in `community_assignment_service_test.go` pass GREEN using mocked dependencies.
2. `community_service.go` depends only on interfaces from `ports/outbound/` — no adapter or pgx imports.
3. `AssignmentUseCase` inbound port compiles with the updated `Assign` and new `ListByCommunity` signatures.
4. `CommunityAssignmentRepository` outbound port is defined and imports only domain model types.
