# TASK-M6.0-T8: Agent NATS Inbound Adapter & Offloading

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T8                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Agent Adapters — `internal/agent/adapters/inbound/nats` |
| Status      | IMPLEMENTED                                             |
| Depends On  | TASK-M6.0-T7                                            |

## Objective

Redesign the agent NATS subscriber from a named echo subscriber to a generic event subscriber that dispatches events using the `X-Tacito-Schema` header. Extract S3 payload offloading to a shared utility helper.

## Files

| File | Action |
|------|--------|
| `internal/agent/adapters/inbound/nats/event_subscriber.go` | NEW |
| `internal/agent/adapters/inbound/nats/event_subscriber_test.go` | NEW |
| `internal/agent/adapters/inbound/nats/offload.go` | NEW |

## RED Phase

Write unit tests in `internal/agent/adapters/inbound/nats/event_subscriber_test.go` and offload tests:

- `TestEventSubscriber_Routing`: Mock `SchemaRouter`; construct a message with header `X-Tacito-Schema` set to conversational schema URNs; verify the subscriber receives it, decodes it into `DomainEvent`, and routes it via `SchemaRouter.Route`.
- `TestEventSubscriber_InvalidSchema`: Verify that when message lacks schema header or has invalid schema header, it is logged and dropped without calling `SchemaRouter.Route`.
- `TestOffloadUtility`: Assert that `offload.go` functions `NormalizeBucketName`, `unescapeReader`, and upload operations correctly upload payloads > 256KB to the configured S3 blob store and generate an `s3_reference` JSON payload reference string.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/agent/adapters/inbound/nats/offload.go`:
   - Move bucket name normalization, `unescapeReader`, and `S3Reference` structures.
   - Define a helper to check and offload large payload message contents to `BlobStore`.
2. Create `internal/agent/adapters/inbound/nats/event_subscriber.go`:
   - Implement `EventSubscriber` listening to `ts.community.{communityID}.agent.{agentName}`.
   - Extract tracing span context from `Traceparent` NATS header and propagate it to the handler context.
   - Dispatch decoded events to `SchemaRouter.Route`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm tracing context extraction complies with W3C propagation standards.
- Verify offload threshold (256KB) is explicitly checked, and error paths do not leave unclosed files or memory leaks.
