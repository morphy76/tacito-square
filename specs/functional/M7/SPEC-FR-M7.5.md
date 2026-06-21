# SPEC-FR-M7.5: Standardize query parameters for GET APIs

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.5                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | keeper, bff                                 |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

Standardize query parameters for GET APIs (excluding specific lists like agent cards) to accept standardized query parameters for model filtering, sorting, paginating, and timezone/timestamp formats.

## Specification

1. All GET list endpoints (excluding Agent Cards) MUST support a standardized query parameter structure.
2. The parameters MUST include standard sorting (`sort_by`, `sort_dir`), pagination (`page`, `limit`), and filtering fields.
3. Timezone/timestamp formats in query filters and response models MUST follow ISO 8601 formatting.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
