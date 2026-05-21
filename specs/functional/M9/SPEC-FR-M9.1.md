# SPEC-FR-M9.1: A2A HTTP Gateway

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.1                                |
| Status        | DRAFT                                       |
| Milestone     | M9                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M6.5                                |
| Supersedes    | none                                        |

## Context

External agents communicate with internal Tacito Square agents via an A2A HTTP gateway exposed by keeper.

## Specification

1. Keeper MUST expose A2A protocol HTTP endpoints at `/a2a/v1/`.
2. External agents MUST present valid Agent Cards for registration.
3. Inbound A2A messages MUST be relayed to the appropriate internal agent via NATS.
4. Outbound responses MUST be streamed back to the external agent.
5. The gateway MUST enforce rate limiting on external requests (per SPEC-NFR-CLOUD).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
