# TASK-M4.8-T2: Application Ports — EchoUseCase & CommunityBroadcaster

| Field       | Value                                                   |
|-------------|---------------------------------------------------------|
| Task ID     | TASK-M4.8-T2                                            |
| Spec        | SPEC-FR-M4.8                                            |
| Boundary    | Application Ports — `internal/keeper/application/ports` |
| Status      | TODO                                                    |
| Depends On  | TASK-M4.8-T1                                            |

## Objective

Define the two new port interfaces that bracket the echo use case:
- `EchoUseCase` — the **inbound driving port** consumed by the HTTP handler.
- `CommunityBroadcaster` — the **outbound driven port** isolating the application service from the NATS infrastructure.

These are pure interface definitions with no implementation; they exist solely to enforce the hexagonal boundary.

## Files

| File | Action |
|------|--------|
| `internal/keeper/application/ports/inbound/usecases.go` | MODIFY |
| `internal/keeper/application/ports/outbound/community_broadcaster.go` | NEW |

## RED Phase

Since these are interface definitions, the RED phase is satisfied by the compile failures in T3 (NATS adapter) and T4 (application service) that depend on these interfaces. However, add a compile-time assertion test:

Create `internal/keeper/application/ports/outbound/community_broadcaster_test.go`:

```go
package outbound_test

import (
    "testing"
    "github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

// TestCommunityBroadcasterIsInterface asserts the interface is reachable.
// Real behaviour is tested in the adapter and service layers.
func TestCommunityBroadcasterIsInterface(t *testing.T) {
    var _ outbound.CommunityBroadcaster = (outbound.CommunityBroadcaster)(nil)
}
```

Run `make test` — fails because the file doesn't exist yet (RED).

## GREEN Phase

**Extend `internal/keeper/application/ports/inbound/usecases.go`** — append at the end:

```go
// EchoUseCase defines the driving port for the community echo feature.
type EchoUseCase interface {
    // EchoCommunity sanitizes the message, fans it out to all running agents in
    // the community via NATS request-reply, and returns the aggregated results.
    EchoCommunity(ctx context.Context, communityID uuid.UUID, message string) (*model.CommunityEchoResponse, error)
}
```

Add import `"github.com/morphy76/tacito-square/internal/keeper/domain/model"` to the existing import block if not already present.

**Create `internal/keeper/application/ports/outbound/community_broadcaster.go`**:

```go
package outbound

import (
    "context"
    "github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

// CommunityBroadcaster is the driven outbound port for per-agent NATS request-reply.
// The subject format used by implementations is: ts.community.{communityID}.agent.{agentName}
type CommunityBroadcaster interface {
    // RequestEcho sends an EchoRequest to the given agent's NATS subject and
    // blocks until it receives an EchoReply or the context deadline is exceeded.
    // Returns an error on timeout or marshal/unmarshal failure.
    RequestEcho(ctx context.Context, communityID, agentName string, req model.EchoRequest) (*model.EchoReply, error)

    // Available reports whether the underlying NATS connection is established.
    // The application service MUST check this before fanning out requests.
    Available() bool
}
```

Run `make test` — compile assertions pass (GREEN).

## REFACTOR Phase

- Confirm `EchoUseCase` is added consistently alongside the other use case interfaces in `usecases.go`.
- Confirm `CommunityBroadcaster` does not import any concrete `nats.go` types — only domain model types and standard library.
- Confirm `Available()` method is documented as a non-blocking connectivity check.
