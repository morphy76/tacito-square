# SPEC-FR-M6.2: NATS JetStream Integration & At-Least-Once Delivery

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.2                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | agent, keeper, bff                          |
| Depends On    | SPEC-FR-M6.0                                |
| Supersedes    | none                                        |

## Context

While standard NATS pub/sub is implemented for transient, fire-and-forget event routing (e.g., heartbeats and real-time streaming), critical business events—such as starting conversation threads, user messages, and final agent responses—require at-least-once delivery guarantees, message persistence, and durable subscriber state. 

This specification defines the migration of critical event subjects to NATS JetStream.

## Specification

1. **JetStream Stream Configuration**:
   - The system MUST define and configure a JetStream stream named `TACITO_EVENTS` covering subject pattern `ts.community.>`.
   - The stream retention policy MUST be configured with limits-based retention (e.g., max age of 7 days, max message count, or max bytes capacity).
   - Storage type MUST be configurable (file-based in production, in-memory in testing).

2. **Durable Consumers**:
   - For agent components, the system MUST utilize durable pull/push consumers to process events sequentially per thread or load-balanced across replicas via NATS Queue Groups.
   - For the keeper and conversation-hub components, durable consumers MUST be used to reliably persist events even during service restarts.

3. **Message Acknowledgements (ACKs)**:
   - Subscribers processing events under critical schemaRefs (e.g., `start-thread`, `add-user-message`, `end-thread`, `agent-response`) MUST explicitly acknowledge (`Msg.Ack()`) the message *only* after processing completes successfully.
   - Failed processing MUST result in a negative acknowledgement (`Msg.Nak()`) to trigger redelivery.

4. **Redelivery & Backoff Policy**:
   - Configure a maximum redelivery attempt limit (e.g., 5).
   - Implement exponential backoff or linear delays between redeliveries to prevent loop pressure.
   - Unackable messages exceeding maximum attempts MUST be routed to a Dead Letter Queue (DLQ) subject (e.g., `ts.dlq.community.>`).

5. **Deduplication**:
   - NATS JetStream deduplication MUST be enabled on the stream, matching on the `X-Tacito-Event-ID` message header.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.

