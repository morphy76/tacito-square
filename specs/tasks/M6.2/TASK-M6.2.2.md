# TASK-M6.2.2: NATS JetStream Durable Pull Consumers & DLQ Routing

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M6.2.2                                 |
| Status        | IMPLEMENTED                                 |
| Spec          | SPEC-FR-M6.2                                |
| Depends On    | TASK-M6.2.1                                 |

## Description

Migrate the agent's NATS subscriber from standard push subscriptions to Durable Pull Consumers and implement manual Dead Letter Queue (DLQ) routing for processing failures.

## Work Items

1. **RED Phase**:
   - Update `internal/agent/adapters/inbound/nats/event_subscriber_test.go` to:
     - Simulate JetStream message delivery.
     - Verify that a subscriber fetches messages via the pull API.
     - Verify that if message processing fails 5 times, it gets republished to the DLQ subject (`ts.dlq.community.>`) and is then acked.
     - Ensure these tests fail as expected on the current codebase.

2. **GREEN Phase**:
   - Update `internal/agent/adapters/inbound/nats/event_subscriber.go`:
     - Initialize JetStream context.
     - Define and assert the `TACITO_EVENTS` and `TACITO_DLQ` streams on start.
     - Replace `Subscribe` and `QueueSubscribe` calls with a Durable Pull Consumer subscription (`js.PullSubscribe` or equivalent API).
     - Run a background go-routine for pull consumer fetching loop that fetches messages when there is thread capacity.
     - In the message handler, check the delivery count of the message. If it exceeds 5, publish to the DLQ subject and call `msg.Ack()`. Otherwise, process the message and `msg.Ack()` on success, or `msg.Nak()` on failure.

3. **REFACTOR Phase**:
   - Abstract the NATS JetStream client loop cleanly to avoid goroutine leaks.
   - Standardize error handling and structured logs with `zerolog`.

## Acceptance Criteria

1. Agent consumes community events using Durable Pull Consumers.
2. Messages failing 5 times are routed to `TACITO_DLQ` and acked from the source stream.
3. Test suite for the subscriber is completely GREEN.
