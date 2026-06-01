# SPEC-FR-M8.5: HITL Yield & Callback Flows

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.5                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | agent, keeper                               |
| Depends On    | SPEC-FR-M5.2, SPEC-FR-M6.4                 |
| Supersedes    | none                                        |

## Context

Agents can yield to a human during reasoning for input, approval, or clarification. HITL pauses reasoning, notifies the human, and resumes on response.

## Specification

1. An agent MUST yield to a human by emitting a HITL yield event via NATS.
2. Keeper MUST persist the HITL callback in PostgreSQL.
3. The callback MUST be exposed via `GET /api/v1/threads/{id}/hitl` and `POST /api/v1/threads/{id}/hitl/{callback_id}/respond`.
4. Keeper MUST relay the human response to the agent via NATS.
5. The agent MUST resume reasoning with the human response as additional context.
6. Unanswered callbacks MUST escalate after configurable TTL (default: 1 hour).
7. The Agent Card MUST indicate HITL capability via a `humanInTheLoop` flag.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
