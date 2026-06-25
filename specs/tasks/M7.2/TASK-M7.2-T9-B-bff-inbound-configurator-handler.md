# TASK-M7.2-T9-B: BFF Inbound HTTP Configurator Handler, Auth & Composition

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T9-B                                                     |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | BFF Inbound (`internal/bff/adapters/inbound/http/`), Bootstrapping (`internal/bff/`) |
| Status      | DRAFT                                                              |
| Depends On  | TASK-M7.2-T9-A                                                     |

## Objective

Modify session middleware to propagate the tenant context, enforce admin/spawner role validation on configurator endpoints, and implement the configurator routes including parallel endpoint composition.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/session_middleware.go` | MODIFY |
| `internal/bff/adapters/inbound/http/configurator_handler.go` | NEW |
| `internal/bff/adapters/inbound/http/configurator_handler_test.go` | NEW |
| `internal/bff/adapters/inbound/http/routes.go` | MODIFY |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |

## RED Phase

1. **Write Configurator Bridging & Composition Tests**:
   - Create `internal/bff/adapters/inbound/http/configurator_handler_test.go`.
   - Write unit tests verifying:
     - **Role Authorization Denied**: A request to configurator routes with a valid session cookie but containing *no* required roles (e.g. only `viewer`) is rejected with `403 Forbidden` and error payload `{ "error": "Insufficient permissions" }`.
     - **Role Authorization Allowed**: A request with `keeper-admin` or `agent-spawner` role succeeds.
     - **Tenant Propagation Check**: The outbound call to Keeper includes the `X-Tenant-ID` header matching the user's `TenantID`.
     - **API Composition endpoint**: A `GET /api/v1/configurator/wizard/options` triggers parallel backend requests to the Keeper for LLM bindings, skills, and prompts, combining them into a single coherent options dictionary response.
   - Run tests (`make test`) — confirm they fail.

## GREEN Phase

1. **Update Session Middleware to Propagate Tenant Context**:
   - In `internal/bff/adapters/inbound/http/session_middleware.go`, import the `tenant` package (`github.com/morphy76/tacito-square/internal/shared/tenant`).
   - In `enrichContext`, call `tenant.ContextWithTenant` using the parsed `userInfo.TenantID` to set the tenant context on the request context (`c.Request.Context()`). This allows downstream components and outbound adapters to access the tenant context.

2. **Implement Configurator Handler & Composite Endpoints**:
   - Create `internal/bff/adapters/inbound/http/configurator_handler.go`.
   - Implement `GetWizardOptions` to call `GetLLMBindings`, `GetSkills`, and `GetPrompts` from Keeper in parallel using goroutines/channels and group the results.
   - Implement the remaining configuration bridge routes (CRUD for agents, communities, assignments).
   - Ensure the new routes are registered under the `/configurator` group in `internal/bff/adapters/inbound/http/routes.go`.
   - Update `internal/bff/bootstrap.go` to initialize and wire the configurator handler.
   - Run `make test` and confirm all backend tests pass.

## REFACTOR Phase

- Optimize parallel execution in API composition to handle downstream timeouts gracefully using Go contexts.
- Clean up response error formatting to ensure Keeper domain errors are translated into readable user-facing error messages.
- Audit tracing context propagation to ensure that parent trace spans encompass the parallel composite requests.
