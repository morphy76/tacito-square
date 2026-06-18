# SPEC-FR-M6.2: NATS JetStream Integration & At-Least-Once Delivery

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.2                                |
| Status        | ACCEPTED                                    |
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
   - The system MUST define and configure a Dead Letter Queue stream named `TACITO_DLQ` covering subject pattern `ts.dlq.community.>`.
   - The stream retention policy MUST be configured with limits-based retention (e.g., max age of 7 days, max message count, or max bytes capacity).
   - Storage type MUST be configurable (file-based in production, in-memory in testing).

2. **Durable Consumers**:
   - Standardize on NATS JetStream **Durable Pull Consumers** (using the new Go NATS JetStream client API `jetstream.New(nc)`) for all background event consumers in the `agent` and `keeper` components.
   - Multiple active replicas of a component MUST subscribe to the same named Durable Pull Consumer, allowing NATS to automatically load-balance messages across replicas with backpressure flow control (fetching only when replica active threads permit).

3. **Message Acknowledgements (ACKs) & DLQ**:
   - Subscribers processing events under critical schemaRefs (e.g., `start-thread`, `add-user-message`, `end-thread`, `agent-response`) MUST explicitly acknowledge (`Msg.Ack()`) the message *only* after processing completes successfully.
   - Failed processing MUST result in a negative acknowledgement (`Msg.Nak()`) to trigger redelivery.
   - When a consumer receives a message where the metadata delivery count (`msg.Metadata.NumDelivered` or equivalent) exceeds the maximum limit (5 attempts), the subscriber MUST manually publish the payload to the corresponding DLQ subject `ts.dlq.community.<id>...` and then explicitly acknowledge (`Msg.Ack()`) the original message.

4. **Redelivery & Backoff Policy**:
   - Configure a maximum redelivery attempt limit of 5.
   - Configure NATS consumer backoff schedule parameters via Viper configuration to prevent loop pressure.

5. **Deduplication**:
   - NATS JetStream stream-level deduplication MUST be enabled, matching on the standard NATS header `Nats-Msg-Id`.
   - Outbound NATS event publishers MUST copy/map the application event's unique ID (`X-Tacito-Event-ID` / `EventID`) to the `Nats-Msg-Id` header.

6. **Publisher ACKs**:
   - Outbound NATS event publishers (`NATSEventPublisher` in both `agent` and `keeper` components) MUST await JetStream publish acknowledgements with context timeout enforcement when publishing to stream subjects (e.g., matching `ts.community.>`).
   - Standard transient, non-stream events (such as heartbeats and agent announcements) MUST continue to use standard async NATS pub/sub, bypassing JetStream blocking ACKs.

## Acceptance Criteria

- **AC1: Stream Provisioning**: The `TACITO_EVENTS` and `TACITO_DLQ` streams are automatically created/asserted on NATS client initialization if not present, with parameters loaded from the configuration file.
- **AC2: Publisher ACKs**: Outbound publishers block and await JetStream publish acknowledgement for stream events. Timeout or connection loss triggers appropriate errors.
- **AC3: Deduplication**: Duplicate publications (messages with identical `Nats-Msg-Id` within the duplication window) are discarded by the JetStream stream natively.
- **AC4: Load-Balanced Consumers**: Durable Pull Consumers allow multiple agent replicas to fetch messages round-robin. Message ordering is maintained sequentially per thread via the existing Redis-based Distributed Lock (`RedisMemoryAdapter`).
- **AC5: DLQ Routing**: Messages failing processing 5 times are published to the `ts.dlq.community.>` stream and successfully acknowledged on the main stream.
- **AC6: Heartbeat Bypass**: Non-stream events like heartbeats and announcements are published using standard async NATS pub/sub without blocking on JetStream ACKs.

## Test Plan

- **Integration Tests**:
  - Run a local ephemeral NATS Server with JetStream enabled via `testcontainers-go`.
  - Verify stream auto-provisioning.
  - Verify that sending duplicates with the same `Nats-Msg-Id` results in deduplication.
  - Verify that a subscriber failing to process a message nak's it, and after 5 failed deliveries, the message is manually published to the `TACITO_DLQ` stream and acked.
- **Manual Verification**:
  - Deploy to the local Kubernetes cluster and use NATS CLI tools (`nats stream info`, `nats consumer info`) to verify active stream capacities, limits, and consumer parameters.

## Files Affected

- `tools/helm/tacito-square-infra/values.yaml` (Enable JetStream in NATS subchart configuration)
- `internal/keeper/adapters/outbound/nats/event_publisher.go` (Use JetStream publish & map header)
- `internal/agent/adapters/outbound/nats/publisher.go` (Use JetStream publish & map header)
- `internal/agent/adapters/inbound/nats/event_subscriber.go` (Migrate to Durable Pull Consumers)
- `internal/keeper/adapters/outbound/nats/event_subscriber.go` (Migrate to Durable Pull Consumers)
- `internal/agent/adapters/inbound/nats/event_subscriber_test.go` (Update tests to assert JetStream pull consumer and redelivery)
- `internal/keeper/adapters/outbound/nats/event_publisher_test.go` (Update tests to mock/assert JetStream publish)


