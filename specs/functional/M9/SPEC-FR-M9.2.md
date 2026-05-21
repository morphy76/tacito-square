# SPEC-FR-M9.2: External Agent Registry

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.2                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M9.1                                |
| Supersedes    | none                                        |

## Context

External A2A agent sources must be registered and health-monitored for reliable federation.

## Specification

1. Registry API: `GET/POST /api/v1/external-sources` and `GET/PUT/DELETE /api/v1/external-sources/{id}`.
2. Each source MUST include: endpoint URL, Agent Card URL, health check interval.
3. The system MUST periodically poll external source health endpoints.
4. Circuit breakers MUST disable unhealthy sources (per SPEC-NFR-CLOUD).
5. External source status MUST be visible in the Auditor UI.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
