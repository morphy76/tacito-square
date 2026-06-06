# TASK-M6.0-T2: Keeper Application Ports

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M6.0-T2                                            |
| Spec        | SPEC-FR-M6.0                                            |
| Boundary    | Keeper Ports — `internal/keeper/application/ports`      |
| Status      | DRAFT                                                   |
| Depends On  | TASK-M6.0-T1                                            |

## Objective

Define the new inbound and outbound ports for the keeper:
- Outbound: `EventPublisher` (to publish events to NATS).
- Outbound: `EventSubscriber` and `EventSubscription` (to subscribe to event stream wildcard).
- Inbound: `EventUseCase` (to publish events on behalf of HTTP client).
- Inbound: `EventStreamUseCase` (to stream events to HTTP client).
- Inbound: Delete/cleanup `EchoUseCase`.

## Files

| File | Action |
|------|--------|
| `internal/keeper/application/ports/outbound/event_publisher.go` | NEW |
| `internal/keeper/application/ports/outbound/event_subscriber.go` | NEW |
| `internal/keeper/application/ports/inbound/usecases.go` | MODIFY |
| `internal/keeper/application/ports/outbound/community_broadcaster.go` | DELETE |

## RED Phase

Create compile-time assertion tests to verify the interfaces exist and are importable:

Create `internal/keeper/application/ports/outbound/ports_test.go`:
```go
package outbound_test

import (
	"testing"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

func TestKeeperPortsReachable(t *testing.T) {
	var _ outbound.EventPublisher = (outbound.EventPublisher)(nil)
	var _ outbound.EventSubscriber = (outbound.EventSubscriber)(nil)
}
```

Run `make test` — must fail to compile or find files (RED).

## GREEN Phase

1. Create `internal/keeper/application/ports/outbound/event_publisher.go` defining the `EventPublisher` interface matching Section 3.3.
2. Create `internal/keeper/application/ports/outbound/event_subscriber.go` defining `EventSubscriber` and `EventSubscription` interfaces matching Section 4.2.
3. Modify `internal/keeper/application/ports/inbound/usecases.go` to:
   - Import `github.com/morphy76/tacito-square/pkg/events` and `github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound`
   - Delete the `EchoUseCase` interface definition
   - Declare `EventUseCase` with `PublishEvent(ctx context.Context, schemaRef string, payload []byte) (events.DomainEvent, error)`
   - Declare `EventStreamUseCase` with `SubscribeEvents(ctx context.Context, tenantID string, handler func(*events.DomainEvent)) (outbound.EventSubscription, error)`
4. Delete `internal/keeper/application/ports/outbound/community_broadcaster.go`.

Run `make test` — compile assertions pass (GREEN).

## REFACTOR Phase

- Confirm ports contain no concrete infrastructural imports (e.g. no `github.com/nats-io/nats.go`).
- Verify interfaces are clean, readable, and properly documented.
