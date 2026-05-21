# SPEC-FR-M6.3: NATS Subject Namespacing

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.3                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent, keeper                               |
| Depends On    | SPEC-FR-M6.2                                |
| Supersedes    | none                                        |

## Context

NATS subjects must be namespaced to isolate communities and prevent unauthorized cross-community communication.

## Specification

1. NATS subjects MUST follow the format `ts.community.{community_id}.agent.{agent_id}`.
2. Community broadcast MUST use wildcard `ts.community.{community_id}.agent.*`.
3. Keeper monitoring subjects MUST use `ts.keeper.{agent_id}` for housekeeping.
4. Agents MUST only subscribe to subjects within their assigned community.
5. Subject authorization SHOULD be enforced via NATS account configuration.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
