# TASK-REFACTOR-reconcile_service: Refactor reconcile_service.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-reconcile_service            |
| Status        | VERIFIED                                    |
| Target File   | [reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go)  |
| Baseline Tests| All existing tests MUST pass without changes |

## Description
Split manifest-building functions and default constants out of `reconcile_service.go` into a new file `reconcile_manifests.go` to reduce structural complexity and file length.

## Work Items
1. **Baseline Phase**:
   - [x] Verify all existing tests pass.
2. **Refactor Phase**:
   - [x] Create a new file [reconcile_manifests.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_manifests.go) and move manifest-building logic (`BuildDeployment`, `BuildHeadlessService`, `getAgentSetting`, `buildOwnerReference`, `boolPtr`) and fallback default constants to it.
   - [x] Keep only the core `Reconcile` state machine, constructor, and `LoadTierMap` inside [reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go).
3. **Verification Phase**:
   - [x] Run existing tests to ensure they are 100% green.
   - [x] Run `make lint` to confirm codebase styling compliance.

## Acceptance Criteria
1. No existing unit/integration/contract test is modified.
2. [reconcile_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/operator/application/service/reconcile_service.go) has its LOC reduced from 621 lines to under 260 lines.
3. All existing HTTP and controller tests remain fully green.
4. Lint checks pass cleanly.
