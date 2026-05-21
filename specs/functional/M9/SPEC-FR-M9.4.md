# SPEC-FR-M9.4: OpenAPI Contract Validation

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.4                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | all                                         |
| Depends On    | SPEC-NFR-OPENAPI                            |
| Supersedes    | none                                        |

## Context

API contracts must be validated to ensure the running system matches committed OpenAPI specifications.

## Specification

1. Each component MUST serve its OpenAPI spec at `GET /openapi.json`.
2. Contract tests MUST compare live spec against committed spec in `api/openapi/`.
3. Contract test failures MUST block CI.
4. The system SHOULD provide Swagger UI at `/swagger/` in dev mode.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
