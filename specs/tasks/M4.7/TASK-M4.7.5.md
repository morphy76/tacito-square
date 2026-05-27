# TASK-M4.7.5: OpenAPI Documentation & Contract Tests

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M4.7.5                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M4.7                                |
| Depends On    | TASK-M4.7.4                                 |

## Description

Document the 6 new lifecycle REST API endpoints in the project's OpenAPI specification (`api/openapi/openapi.json`). All endpoint paths, parameters, payloads, and response codes (including the complex `207 Multi-Status` JSON responses) must be fully and accurately described. Run existing contract tests to verify zero drift between Gin handler registrations and the OpenAPI specifications.

## Boundary & Target Functions

- **Component**: `api`, `test`
- **Files**:
  - `api/openapi/openapi.json`
- **Target Operations**:
  - Add tag `lifecycle/management` to the top-level `tags` list.
  - Add path registrations:
    - `/api/v1/agents/{agent_id}/deploy`
    - `/api/v1/agents/{agent_id}/undeploy`
    - `/api/v1/agents/{agent_id}/status`
    - `/api/v1/communities/{community_id}/deploy`
    - `/api/v1/communities/{community_id}/undeploy`
    - `/api/v1/communities/{community_id}/status`

## Work Items

1. **RED Phase**:
   * Add empty or stub paths to `api/openapi/openapi.json` and run the contract test suite (e.g. `make test-contract`).
   * Confirm that the tests fail (RED) due to the mismatch between registered Gin endpoints and the OpenAPI contract.

2. **GREEN Phase**:
   * Add complete and valid OpenAPI definitions for all 6 endpoints in `api/openapi/openapi.json`.
   * Under the community deploy and undeploy routes, detail the `207` schema, including the status results structure per agent.
   * Tag all operations strictly with `lifecycle/management`.
   * Run the contract test suite and verify that the tests compile and run successfully (GREEN).

3. **REFACTOR Phase**:
   * Review all OpenAPI description descriptions to make sure they match standard REST naming conventions and remain clear, readable, and consistent.

## Acceptance Criteria

1. Contract tests pass successfully with no schema mismatch.
2. The OpenAPI JSON document validates cleanly against the OpenAPI 3.x specification.
3. Every operation has exactly one tag matching `domain/subdomain` format (`lifecycle/management`).
