# SPEC-FR-M6.6: Conversation Handoff

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.6                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M6.2, SPEC-FR-M5.3                 |
| Supersedes    | none                                        |

## Context

During reasoning, an agent may determine that another agent is better suited. Handoff transfers conversation context (including STM) to the target agent.

## Specification

1. An agent MUST be able to initiate a handoff via NATS message to the target agent.
2. The handoff message MUST include: thread ID, conversation context, reason, source agent ID.
3. The target agent MUST acknowledge before the source releases the thread.
4. STM context MUST be migrated to the target agent's Redis keyspace.
5. A handoff event MUST be emitted for audit purposes.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
