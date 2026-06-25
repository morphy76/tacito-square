# TASK-M7.2-T9-A: BFF Outbound Keeper Client & Contract Testing

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T9-A                                                     |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | BFF Outbound Client (`internal/bff/application/ports/outbound/`, `internal/bff/adapters/outbound/`), Test (`test/contract/`) |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.1, TASK-M7.2-T7, TASK-M7.2-T8                           |

## Objective

Extend the `KeeperClient` outbound port and implement its `KeeperHTTPClient` adapter to cover all CRUD operations for the Keeper resources (agents, communities, assignments, prompts, skills, LLM bindings, and MCP clients). Add a contract-based test suite verifying request/response payload compatibility against `api/openapi/openapi.json` to prevent integration drift.

## Files

| File | Action |
|------|--------|
| `internal/bff/application/ports/outbound/keeper_client.go` | MODIFY |
| `internal/bff/adapters/outbound/keeper_http_client.go` | MODIFY |
| `test/contract/bff_keeper_contract_test.go` | NEW |

## RED Phase

1. **Write BFF-Keeper Client Contract Test**:
   - Create `test/contract/bff_keeper_contract_test.go`.
   - Write tests that parse `api/openapi/openapi.json` and verify that the `KeeperHTTPClient` adapter requests match the paths, methods, parameters, and request/response body schemas defined in the OpenAPI specification.
   - Run tests (`make test-contract`) — verify that the contract tests fail due to missing operations on `KeeperClient` / `KeeperHTTPClient`.

## GREEN Phase

1. **Build the OpenAPI-Driven Keeper Client Adapter**:
   - Update the `KeeperClient` interface under `internal/bff/application/ports/outbound/keeper_client.go` to declare operations: `GetAgents`, `CreateAgent`, `UpdateAgent`, `DeleteAgent`, `GetCommunities`, `CreateCommunity`, etc., matching the DTO schemas defined in the Keeper's `api/openapi/openapi.json`.
   - Update `internal/bff/adapters/outbound/keeper_http_client.go` to implement these operations, mapping endpoints and parsing payloads to matching Go structures.
   - Inject context-based tenant information: extract the active tenant ID from the context via `tenant.FromContext(ctx)` and add it as the `X-Tenant-ID` header on all outbound Keeper requests.
   - Ensure `make test-contract` passes.

## REFACTOR Phase

- Clean up client request/response helper functions (e.g. shared dynamic JSON encoding, response parsing, and standard HTTP error wrapping).
