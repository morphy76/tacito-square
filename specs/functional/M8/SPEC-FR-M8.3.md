# SPEC-FR-M8.3: Quota Enforcement (Redis counters)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.3                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M8.2, SPEC-NFR-CACHE               |
| Supersedes    | none                                        |

## Context

Quotas are enforced in real-time using Redis atomic counters for low-latency checks.

## Specification

1. The system MUST use Redis atomic counters for real-time quota tracking.
2. Counter keys MUST follow `ts:keeper:quota:{type}:{id}:{period}`.
3. Message quota checks MUST be performed before agent message delivery.
4. LLM token quota checks MUST be performed before LLM API calls.
5. Quota exceeded responses MUST return HTTP 429 with retry-after header.
6. Counters MUST reset automatically using Redis TTL.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
