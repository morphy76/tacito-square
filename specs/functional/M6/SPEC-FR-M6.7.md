# SPEC-FR-M6.7: Specialist Agent Spawn

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.7                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | keeper, agent                               |
| Depends On    | SPEC-FR-M6.1, SPEC-FR-M3.6                 |
| Supersedes    | none                                        |

## Context

A hub agent may request keeper to spawn a specialist agent for a specific sub-task. The specialist joins the community dynamically.

## Specification

1. A hub agent MUST request specialist spawning via NATS message to keeper.
2. The spawn request MUST include: agent definition reference, community ID, reason.
3. Keeper MUST validate against community quotas before proceeding.
4. Keeper MUST create a TacitoAgent CRD for the specialist.
5. The specialist MUST join the community and announce via Agent Card.
6. Specialist agents MAY auto-terminate after task completion.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
