# SPEC-FR-M8.2: Usage Quotas (community + agent)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.2                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.6                                |
| Supersedes    | none                                        |

## Context

Communities and agents have configurable usage limits to prevent resource exhaustion.

## Specification

1. The system MUST define a `Quota` value object: maxAgents (per community), maxMessages (per agent/hour), maxLLMTokens (per agent/day).
2. Quotas MUST be configurable per community and per agent definition.
3. The system MUST expose quota CRUD via `GET/PUT /api/v1/communities/{id}/quotas` and `GET/PUT /api/v1/agents/{id}/quotas`.
4. Default quotas MUST be applied when none are explicitly configured.
5. Quota configuration MUST be persisted in PostgreSQL.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
