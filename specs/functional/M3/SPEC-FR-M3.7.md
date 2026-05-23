# SPEC-FR-M3.7: Agent-Community Assignment

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.7                                |
| Status        | DRAFT                                       |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.5, SPEC-FR-M3.6                 |
| Supersedes    | none                                        |

## Context

Agents exist as definitions but their runnable instances are always bound to a community. This spec covers the assignment of agent definitions to communities, which triggers CRD creation and ultimately agent pod instantiation by the operator.

## Specification

1. The system MUST expose an assignment endpoint: `POST /api/v1/communities/{community_id}/agents/{agent_id}`.
2. An agent definition MAY be assigned to multiple communities (each assignment creates a separate runnable instance).
3. Assignment MUST validate that both the agent definition and community exist and are in valid states.
4. Assignment MUST trigger the creation of a TacitoAgent CRD (via SPEC-FR-M3.10).
5. Unassignment (`DELETE /api/v1/communities/{community_id}/agents/{agent_id}`) MUST trigger CRD deletion.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
