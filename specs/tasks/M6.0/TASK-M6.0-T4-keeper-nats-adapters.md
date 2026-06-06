# TASK-M6.0-T4: Keeper NATS Outbound Adapters

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Task ID     | TASK-M6.0-T4                                             |
| Spec        | SPEC-FR-M6.0                                             |
| Boundary    | Keeper Adapters — `internal/keeper/adapters/outbound/nats` |
| Status      | DRAFT                                                    |
| Depends On  | TASK-M6.0-T3                                             |

## Objective

Implement the `EventPublisher` and `EventSubscriber` outbound adapters using NATS. Ensure that header projection, tracing propagation, wildcard subscription (`ts.community.>`), and tenant isolation are fully executed.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/outbound/nats/event_publisher.go` | NEW |
| `internal/keeper/adapters/outbound/nats/event_subscriber.go` | NEW |
| `internal/keeper/adapters/outbound/nats/event_publisher_test.go` | NEW |
| `internal/keeper/adapters/outbound/nats/community_broadcaster.go` | DELETE |
| `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go` | DELETE |

## RED Phase

Create NATS integration tests in `internal/keeper/adapters/outbound/nats/event_publisher_test.go` using an in-process or local NATS test broker:

- `TestNATSEventPublisher_Publish`: Publish a `DomainEvent` and verify that NATS message headers (`X-Tacito-Schema`, `X-Tacito-Source`, `X-Tacito-Tenant`, `X-Tacito-Event-ID`, `X-Tacito-Occurred`) are correctly populated, and trace context propagates via `Traceparent`.
- `TestNATSEventSubscriber_TenantIsolation`: Subscribe using `Subscribe` wildcard; publish events for two different tenants (`tenantA` and `tenantB`); verify the subscription callback only receives the event matching the resolved subscriber tenant.
- `TestNATSEventSubscriber_Cleanup`: Assert calling `Stop()` on `EventSubscription` unsubscribes the NATS listener.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/keeper/adapters/outbound/nats/event_publisher.go`:
   - Implement `EventPublisher` with constructor `NewNATSEventPublisher(nc *nats.Conn)`.
   - Propagate OTel trace state into NATS message headers using W3C traceparent injector.
2. Create `internal/keeper/adapters/outbound/nats/event_subscriber.go`:
   - Implement `EventSubscriber` with constructor `NewNATSEventSubscriber(nc *nats.Conn)`.
   - Implement subscription wrapper supporting wildcard matching and tenant filtering based on NATS headers.
3. Delete `internal/keeper/adapters/outbound/nats/community_broadcaster.go` and its test.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm NATS connections and subscriptions are closed/drained cleanly on shutdown.
- Verify tracing context injection/extraction follows standard OpenTelemetry conventions.
