# TASK-M4.7.4: HTTP Lifecycle Handlers & Route Registration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.7.4                                 |
| Status        | VERIFIED                                    |
| Spec          | SPEC-FR-M4.7                                |
| Depends On    | TASK-M4.7.3                                 |

## Description

Design and implement the HTTP adapter layer for managing Agent and Community lifecycle routes inside the Keeper. This involves creating the concrete driver `LifecycleHandler` using the Gin web framework, extracting the tenant context via the middleware, checking resource ownership, returning appropriate REST status codes (including `207 Multi-Status` for partial community deployment successes/failures), and registering the new lifecycle routing paths within `bootstrap.go`.

## Boundary & Target Functions

- **Packages**: `internal/keeper/adapters/inbound/http`, `internal/keeper`
- **Files**:
  - `[NEW] internal/keeper/adapters/inbound/http/lifecycle_handlers.go`
  - `[MODIFY] internal/keeper/bootstrap.go`
- **Target Structs & Functions**:
  - `LifecycleHandler` (struct)
  - `NewLifecycleHandler(lifecycleUseCase inbound.LifecycleUseCase) *LifecycleHandler`
  - `(h *LifecycleHandler) DeployAgent(c *gin.Context)`
  - `(h *LifecycleHandler) UndeployAgent(c *gin.Context)`
  - `(h *LifecycleHandler) GetAgentStatus(c *gin.Context)`
  - `(h *LifecycleHandler) DeployCommunity(c *gin.Context)`
  - `(h *LifecycleHandler) UndeployCommunity(c *gin.Context)`
  - `(h *LifecycleHandler) GetCommunityStatus(c *gin.Context)`

## Work Items

1. **RED Phase**:
   * Add unit tests in `internal/keeper/adapters/inbound/http/lifecycle_handlers_test.go` utilizing standard `ServeHTTP` routines in `gin.TestMode` verifying:
     * `POST /api/v1/agents/:agent_id/deploy` dispatches to handler, validates tenant, and returns `202 Accepted` on success.
     * Attempting to deploy/undeploy an agent belonging to another tenant returns `404 Not Found` to prevent entity leakage.
     * `POST /api/v1/communities/:community_id/deploy` returns `207 Multi-Status` with the detailed JSON array structure if parallel execution triggers a partial success/failure condition.
     * All endpoints standard error responses match the project's unified JSON structure `{ "error": "Descriptive message" }`.

2. **GREEN Phase**:
   * Create `internal/keeper/adapters/inbound/http/lifecycle_handlers.go` implementing `LifecycleHandler`.
   * Bind the new Gin routes dynamically.
   * Wire the dynamic tenant extraction via `tenant.FromContext(c.Request.Context())` and abort with `401 Unauthorized` if absent.
   * On cross-tenant operations, return `404 Not Found` (same as missing entity logic).
   * Update `internal/keeper/bootstrap.go` to construct `LifecycleService` and `LifecycleHandler` and register the endpoints:
     * `POST /api/v1/agents/:agent_id/deploy`
     * `POST /api/v1/agents/:agent_id/undeploy`
     * `GET /api/v1/agents/:agent_id/status`
     * `POST /api/v1/communities/:community_id/deploy`
     * `POST /api/v1/communities/:community_id/undeploy`
     * `GET /api/v1/communities/:community_id/status`

3. **REFACTOR Phase**:
   * Ensure that the logger context correlates the OTel `trace_id` and `span_id` automatically inside logs.

## Acceptance Criteria

1. Endpoints match the API specifications and routing requirements exactly.
2. Cross-tenant access validation returns `404 Not Found` unconditionally.
3. Tests run in `gin.TestMode` using standard recorder instances.
