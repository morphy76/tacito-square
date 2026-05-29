# TASK-M4.8-T3: NATS CommunityBroadcaster Outbound Adapter

| Field       | Value                                                         |
|-------------|---------------------------------------------------------------|
| Task ID     | TASK-M4.8-T3                                                  |
| Spec        | SPEC-FR-M4.8                                                  |
| Boundary    | Outbound Adapter — `internal/keeper/adapters/outbound/nats`   |
| Status      | TODO                                                          |
| Depends On  | TASK-M4.8-T2                                                  |

## Objective

Implement `NATSCommunityBroadcaster`, the concrete outbound adapter that implements the `CommunityBroadcaster` port using NATS request-reply. This adapter encapsulates all NATS-specific logic and keeps the application layer clean.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/outbound/nats/community_broadcaster.go` | NEW |
| `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go` | NEW |

## RED Phase

Create `internal/keeper/adapters/outbound/nats/community_broadcaster_test.go` using an in-process NATS server (`natsserver/test` package):

- `TestRequestEcho_Success`: Start in-process NATS server. Subscribe a mock agent reply handler on `ts.community.test-comm.agent.agent-alpha`. Call `RequestEcho(ctx, "test-comm", "agent-alpha", req)`. Assert the returned `EchoReply` has `AgentName == "agent-alpha"` and `Decorated` non-empty.
- `TestRequestEcho_Timeout`: Call `RequestEcho` with a context deadline of 100ms and no subscriber listening. Assert returned error is non-nil (timeout).
- `TestRequestEcho_MarshalPayload`: Assert the NATS message body received by the mock handler correctly deserialises into an `EchoRequest` with the expected `Message`, `CommunityID`, `TenantID`.
- `TestAvailable_Connected`: Assert `Available()` returns `true` when `nc.IsConnected()`.
- `TestAvailable_NilConn`: Construct adapter with `nc == nil`. Assert `Available()` returns `false`.

Run `make test` — tests must fail (RED).

## GREEN Phase

Create `internal/keeper/adapters/outbound/nats/community_broadcaster.go`:

```go
package nats

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
    "github.com/morphy76/tacito-square/internal/keeper/domain/model"
    "github.com/nats-io/nats.go"
    "github.com/rs/zerolog"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// NATSCommunityBroadcaster implements outbound.CommunityBroadcaster via NATS request-reply.
type NATSCommunityBroadcaster struct {
    nc     *nats.Conn
    logger zerolog.Logger
}

var _ outbound.CommunityBroadcaster = (*NATSCommunityBroadcaster)(nil)

func NewNATSCommunityBroadcaster(nc *nats.Conn, logger zerolog.Logger) *NATSCommunityBroadcaster {
    return &NATSCommunityBroadcaster{nc: nc, logger: logger}
}

func (b *NATSCommunityBroadcaster) Available() bool {
    return b.nc != nil && b.nc.IsConnected()
}

func (b *NATSCommunityBroadcaster) RequestEcho(ctx context.Context, communityID, agentName string, req model.EchoRequest) (*model.EchoReply, error) {
    subject := fmt.Sprintf(echoSubjectFormat, communityID, agentName)

    payload, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("marshal echo request: %w", err)
    }

    msg, err := b.nc.RequestMsgWithContext(ctx, &nats.Msg{
        Subject: subject,
        Data:    payload,
    })
    if err != nil {
        return nil, fmt.Errorf("nats request to %s: %w", subject, err)
    }

    var reply model.EchoReply
    if err := json.Unmarshal(msg.Data, &reply); err != nil {
        return nil, fmt.Errorf("unmarshal echo reply: %w", err)
    }

    return &reply, nil
}
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm `var _ outbound.CommunityBroadcaster = (*NATSCommunityBroadcaster)(nil)` compile-time guard is present.
- Confirm subject format constant `echoSubjectFormat` matches the pattern in SPEC-FR-M4.8 (`ts.community.{communityID}.agent.{agentName}`) and aligns with SPEC-FR-M6.3.
- Confirm the adapter never imports any application service types — only the port interface and domain model.
- Log at `debug` level on each request/reply cycle, including `subject`, `community_id`, `agent_name`.
