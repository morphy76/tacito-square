# TASK-M7.2-T9: BFF Configurator Bridging & Tenant Propagation

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T9                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | BFF (`internal/bff/`), UI (`ui/configurator/`)                     |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.1, TASK-M7.2-T7, TASK-M7.2-T8                           |

## Objective

Implement secure, tenant-isolated bridging inside the Go BFF between the Configurator React UI and the backend Keeper service. All configuration requests from the React UI target the `/api/v1/configurator/*` route namespace in BFF. 

The BFF must:
1. **Enforce Role Authorization**: Verify that the authenticated session contains the required administrator roles (`keeper-admin` or `agent-spawner`). If neither role is present, return `403 Forbidden` with a standardized JSON error message.
2. **Propagate Tenant Context**: Extract the user's `TenantID` from the OIDC/JWT context and inject it as the `X-Tenant-ID` header when forwarding requests to the Keeper.
3. **OpenAPI-Driven Client Adapter**: Extend the `KeeperClient` interface and its outbound HTTP client implementation (`KeeperHTTPClient`) to implement all CRUD operations for Keeper resources (agents, communities, assignments, prompts, skills, LLM bindings, and MCP clients). Use DTO types aligned with the schemas defined in the Keeper's committed `api/openapi/openapi.json`.
4. **Contract Testing**: Create a contract test suite verifying that the BFF `KeeperHTTPClient` adapter request/response mapping is fully compatible with the schemas and endpoints defined in `api/openapi/openapi.json` to prevent integration drift.
5. **Configurator API Composition**: Expose composite BFF endpoints (e.g. `GET /api/v1/configurator/wizard/options`) that query the Keeper APIs in parallel (such as fetching available LLM bindings, skills, and prompts) to aggregate and serve a single payload containing all dropdown/wizard context required by the frontend configurator wizard.
6. **UI Resource Adaptation**: Adapt the Configurator UI's interfaces, form states, and API clients to match the actual schemas of the Keeper resources (defined in the OpenAPI contract), supporting both the guided Wizard configuration and the raw JSON Advanced Settings editor. Matching the actual schemas is intended functionally, but technically the UI models can be optimized for the UI purpose and mediated by the BFF. So do not expect exact mapping between UI models and Keeper schemas, try to optimize the data transfer for the UI purposes. 

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/session_middleware.go` | MODIFY |
| `internal/bff/application/ports/outbound/keeper_client.go` | MODIFY |
| `internal/bff/adapters/outbound/keeper_http_client.go` | MODIFY |
| `internal/bff/adapters/inbound/http/configurator_handler.go` | NEW |
| `internal/bff/adapters/inbound/http/configurator_handler_test.go` | NEW |
| `test/contract/bff_keeper_contract_test.go` | NEW |
| `internal/bff/adapters/inbound/http/routes.go` | MODIFY |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |
| `ui/configurator/src/` | MODIFY |

## RED Phase

1. **Write BFF-Keeper Client Contract Test**:
   - Create `test/contract/bff_keeper_contract_test.go`.
   - Write tests that parse `api/openapi/openapi.json` and verify that the `KeeperHTTPClient` adapter requests match the paths, methods, parameters, and request/response body schemas defined in the OpenAPI specification.
   - Run tests (`make test-contract`) — verify that the contract tests fail due to missing operations on `KeeperClient` / `KeeperHTTPClient`.

2. **Write Configurator Bridging & Composition Tests**:
   - Create `internal/bff/adapters/inbound/http/configurator_handler_test.go`.
   - Write unit tests verifying:
     - **Role Authorization Denied**: A request to configurator routes with a valid session cookie but containing *no* required roles (e.g. only `viewer`) is rejected with `403 Forbidden` and error payload `{ "error": "Insufficient permissions" }`.
     - **Role Authorization Allowed**: A request with `keeper-admin` or `agent-spawner` role succeeds.
     - **Tenant Propagation Check**: The outbound call to Keeper includes the `X-Tenant-ID` header matching the user's `TenantID`.
     - **API Composition endpoint**: A `GET /api/v1/configurator/wizard/options` triggers parallel backend requests to the Keeper for LLM bindings, skills, and prompts, combining them into a single coherent options dictionary response.
   - Run tests (`make test`) — confirm they fail.

3. **Write React UI Resource Tests**:
   - Write unit/component tests in `ui/configurator/` asserting that forms for Agent and Community creation validate payloads correctly against the actual schema types (e.g. `CreateAgentRequest`, `CreateCommunityRequest`) defined in the Keeper OpenAPI spec.

## GREEN Phase

1. **Update Session Middleware to Propagate Tenant Context**:
   - In `internal/bff/adapters/inbound/http/session_middleware.go`, import the `tenant` package (`github.com/morphy76/tacito-square/internal/shared/tenant`).
   - In `enrichContext`, call `tenant.ContextWithTenant` using the parsed `userInfo.TenantID` to set the tenant context on the request context (`c.Request.Context()`).

2. **Build the OpenAPI-Driven Keeper Client Adapter**:
   - Update the `KeeperClient` interface under `internal/bff/application/ports/outbound/keeper_client.go` to declare operations: `GetAgents`, `CreateAgent`, `UpdateAgent`, `DeleteAgent`, `GetCommunities`, `CreateCommunity`, etc.
   - Update `internal/bff/adapters/outbound/keeper_http_client.go` to implement these operations, mapping endpoints and parsing payloads to matching Go structures.
   - Ensure `make test-contract` passes.

3. **Implement Configurator Handler & Composite Endpoints**:
   - Create `internal/bff/adapters/inbound/http/configurator_handler.go`.
   - Implement `GetWizardOptions` to call `GetLLMBindings`, `GetSkills`, and `GetPrompts` from Keeper in parallel using goroutines/channels and group the results.
   - Implement the remaining configuration bridge routes (CRUD for agents, communities, assignments).
   - Ensure the new routes are registered under the `/configurator` group in `internal/bff/adapters/inbound/http/routes.go`.
   - Update `internal/bff/bootstrap.go` to initialize and wire the configurator handler.
   - Run `make test` and confirm all backend tests pass.

4. **Adapt React UI**:
   - Update the API client in `ui/configurator/src/` to connect to the composite `wizard/options` endpoint.
   - Update all configuration types, validations, and step-by-step Wizard/Advanced settings screens to use the exact properties defined in the Keeper models.
   - Confirm the React app compiles and passes all UI component tests.

## REFACTOR Phase

- Optimize parallel execution in API composition to handle downstream timeouts gracefully using Go contexts.
- Clean up response error formatting to ensure Keeper domain errors are translated into readable user-facing error messages.
- Audit tracing context propagation to ensure that parent trace spans encompass the parallel composite requests.
