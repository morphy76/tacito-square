# TASK-M3.7.3: OpenAPI Contract & CRD Coordinator Integration

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M3.7.3                                 |
| Status        | DRAFT                                       |
| Spec          | SPEC-FR-M3.7                                |
| Depends On    | TASK-M3.7.2                                 |

## Description

Define the REST API OpenAPI contract specs, request/response models, and error structures. Integrate a coordinator hook interface to mock and trigger `TacitoAgent` CRD submissions to the Kubernetes API, preparing for operator reconciliation.

## Work Items

1. **RED Phase**:
   - Create contract tests verifying the `/openapi.json` exposed schema matches the updated schema contracts.
   - Write mocks or interface assertions to ensure the assignment endpoint triggers the CRD Coordinator's `Submit` or `Teardown` calls under positive lifecycle flows.

2. **GREEN Phase**:
   - Update `internal/keeper/openapi.json` to define all assignment endpoints, path parameters, schemas, and responses.
   - Attach the `community/communities` stable tag and verify 100% compliance with `SPEC-NFR-OPENAPI`.
   - Declare the coordinator port interface in `internal/keeper/application/ports/outbound/crd_coordinator.go`:
     - `SubmitAgentCRD(ctx context.Context, agent *domain.Agent) error`
     - `TeardownAgentCRD(ctx context.Context, agent *domain.Agent) error`
   - Integrate a trigger to this outbound port inside the HTTP handler / application service upon successful commit in the database.

3. **REFACTOR Phase**:
   - Validate and structure the OpenAPI JSON payloads to ensure clean tags, correct path parameter naming, and precise JSON field definitions.

## Acceptance Criteria

1. OpenAPI contract tests validate the updated `internal/keeper/openapi.json` successfully.
2. All contract parameters comply with the `domain/subdomain` tag rules of `SPEC-NFR-OPENAPI`.
3. Outbound coordinator hooks are triggered successfully on state transitions in TDD unit assertions.
