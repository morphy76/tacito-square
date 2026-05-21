# SPEC-FR-M7.1: BFF API Bridge Layer

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.1                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M2.6                                |
| Supersedes    | none                                        |

## Context

The BFF aggregates and proxies keeper REST API endpoints for consumption by the Configurator and Auditor UIs.

## Specification

1. The BFF MUST be a Gin-based HTTP server (per SPEC-NFR-HTTP).
2. The BFF MUST proxy keeper API endpoints with path prefix `/api/v1/`.
3. The BFF MUST propagate authentication tokens to keeper API calls.
4. The BFF MUST implement circuit breakers on keeper API calls (per SPEC-NFR-CLOUD).
5. The BFF MUST serve static assets for both UI deployments.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
