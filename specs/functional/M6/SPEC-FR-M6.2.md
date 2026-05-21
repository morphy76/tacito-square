# SPEC-FR-M6.2: NATS Inter-Agent Messaging

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.2                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent                                       |
| Depends On    | SPEC-FR-M5.1                                |
| Supersedes    | none                                        |

## Context

Agents communicate asynchronously via NATS messaging. This is the primary communication channel for agent-to-agent interaction within communities and for keeper-to-agent monitoring/housekeeping.

## Specification

1. The system MUST define a `Messaging` port in the agent domain layer.
2. The system MUST implement a NATS adapter for publish/subscribe messaging.
3. The message envelope MUST include: sender ID, recipient ID, community ID, thread ID, message type, payload, timestamp, trace context.
4. The adapter MUST support at-least-once delivery guarantees using NATS JetStream.
5. Keeper MUST use NATS to send monitoring and housekeeping messages to live agents.
6. All message exchanges MUST propagate OpenTelemetry context.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
