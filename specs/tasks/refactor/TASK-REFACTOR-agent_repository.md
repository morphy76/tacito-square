# TASK-REFACTOR-agent_repository: Refactor agent_repository.go

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-REFACTOR-agent_repository              |
| Status        | VERIFIED                                    |
| Target File   | [agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) |
| Baseline Tests| All existing tests MUST pass without changes |

## Description

The [agent_repository.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_repository.go) file contains business validation logic (community status checks, role constraints, and single-agent / hub-spoke topology validations) inside its `AssignToCommunity` database transaction block. According to Hexagonal Architecture guidelines (`RULE[code-architecture.md]`), all domain validation and business rules must reside in the domain or application service layers, not inside SQL adapters.

This task moves these validations to [agent_service.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go).

---

## CRITICAL RULE: Immutable Tests
> [!IMPORTANT]
> **Existing test files (e.g., [agent_assignment_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_assignment_test.go) or other unit/integration tests) MUST NOT be modified.** All tests must pass cleanly out of the box with zero modifications to test code, verifying that behavior is perfectly preserved.

---

## Proposed Refactoring Steps

### 1. Update `AgentService` Dependencies and Initialization
- **Inject `CommunityRepository`**: Modify the [AgentService](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go) constructor to accept `CommunityRepository` as an outbound port dependency.
- **Update Bootstrap Wiring**: Modify the instantiation in [bootstrap.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/bootstrap.go) to pass `communityRepo` to `service.NewAgentService(...)`.
- **Update Test Mocks**: Modify existing unit/mock tests (e.g., [lifecycle_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/lifecycle_service_test.go) and [agent_service_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service_test.go)) to accommodate the updated constructor signature of `AgentService`.

### 2. Relocate Validations to `AgentService.Assign`
- In [AgentService.Assign](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/application/service/agent_service.go#L61):
  1. Retrieve the community via `CommunityRepository.GetByID(ctx, communityID)`.
     - Check if it exists and belongs to the active tenant.
     - Validate that the community status is `"active"` or `"created"`.
  2. Retrieve the agent via `AgentRepository.GetByID(ctx, agentID)`.
     - Check if it exists and belongs to the active tenant.
     - Validate that the agent is not already assigned to another community (`agent.CommunityID == nil`).
  3. Validate topology constraints:
     - **Single Agent Topology**: List existing community agents. If any agent is already assigned, return an error.
     - **Hub-Spoke Topology**: If the assigning agent is a `"hub"`, list existing community agents. If any agent with `role == "hub"` is already assigned, return an error.

### 3. Simplify `AgentRepository.AssignToCommunity`
- Remove community status checks, topology checks, and role counts from `postgres.AgentRepository.AssignToCommunity`.
- Reduce the database transaction down to a simple check verifying that the agent exists, belongs to the tenant, is not already assigned to another community, and performing the SQL `UPDATE` statement.

---

## Verification Plan

### Automated Tests
- Run `make test` to ensure all existing integration and unit tests are completely green.
- Verify that [agent_assignment_test.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/keeper/adapters/outbound/postgres/agent_assignment_test.go) passes without modifications, confirming that database-level behavior and errors match exactly.

### Linter
- Run `make lint` to verify compliance with styling guidelines.
