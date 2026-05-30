# TASK-M4.8-T6: Agent Echo NATS Subscriber

| Field       | Value                                                           |
|-------------|-----------------------------------------------------------------|
| Task ID     | TASK-M4.8-T6                                                    |
| Spec        | SPEC-FR-M4.8                                                    |
| Boundary    | Agent Inbound Adapter — `internal/agent/adapters/inbound/nats`  |
| Status      | IMPLEMENTED                                                     |
| Depends On  | TASK-M4.8-T1                                                    |

## Objective

Implement the agent-side NATS subscriber that receives `EchoRequest` messages, logs the sanitized message, decorates it, and replies with an `EchoReply`. This is the agent component's first inbound NATS adapter and follows the hexagonal structure of `internal/agent/`.

> **Scope note**: The agent component (SPEC-FR-M5.x) is not yet fully scaffolded. This task creates the echo subscriber adapter within the agent's hexagonal structure. It can be bootstrapped into a minimal agent startup or tested in isolation with an in-process NATS server. Full agent wiring is deferred to M5.

## Files

| File | Action |
|------|--------|
| `internal/agent/adapters/inbound/nats/echo_subscriber.go` | NEW |
| `internal/agent/adapters/inbound/nats/echo_subscriber_test.go` | NEW |

## RED Phase

Create `internal/agent/adapters/inbound/nats/echo_subscriber_test.go` using an in-process NATS server:

- `TestEchoSubscriber_Replies`: Start subscriber with `agentName = "agent-alpha"`, `communityID = "comm-1"`. Publish an `EchoRequest` via `nc.RequestWithContext`. Assert the reply is a valid `EchoReply` with:
  - `AgentName == "agent-alpha"`.
  - `Decorated` matches pattern `[agent:agent-alpha at <RFC3339>] hello`.
  - `Timestamp` is a valid RFC3339 string.
- `TestEchoSubscriber_LogsSanitizedMessage`: Use a captured zerolog buffer. Assert `info` log contains `message = "hello"` (sanitized), `agent_name`, `community_id`, `tenant_id`.
- `TestEchoSubscriber_Stop`: Call `Stop()`; assert subsequent messages are not processed (publish after stop, assert no reply within 100ms).
- `TestEchoSubscriber_MalformedPayload`: Publish a non-JSON payload to the subject. Assert the subscriber does not crash and does not reply (log a `warn`).

Run `make test` — tests must fail (RED).

## GREEN Phase

Create `internal/agent/adapters/inbound/nats/echo_subscriber.go`:

```go
package nats

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/morphy76/tacito-square/internal/keeper/domain/model"
    natsclient "github.com/nats-io/nats.go"
    "github.com/rs/zerolog"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// EchoSubscriber listens for EchoRequest messages on the agent's community subject
// and replies with a decorated EchoReply.
type EchoSubscriber struct {
    nc          *natsclient.Conn
    agentName   string
    communityID string
    tenantID    string
    logger      zerolog.Logger
    sub         *natsclient.Subscription
}

// NewEchoSubscriber constructs a new EchoSubscriber. Call Start() to begin listening.
func NewEchoSubscriber(nc *natsclient.Conn, agentName, communityID, tenantID string, logger zerolog.Logger) *EchoSubscriber {
    return &EchoSubscriber{
        nc:          nc,
        agentName:   agentName,
        communityID: communityID,
        tenantID:    tenantID,
        logger:      logger,
    }
}

// Start subscribes to the agent's echo subject. Returns an error if subscription fails.
func (s *EchoSubscriber) Start(_ context.Context) error {
    subject := fmt.Sprintf(echoSubjectFormat, s.communityID, s.agentName)
    sub, err := s.nc.Subscribe(subject, s.handleEcho)
    if err != nil {
        return fmt.Errorf("echo subscriber: subscribe to %s: %w", subject, err)
    }
    s.sub = sub
    s.logger.Info().Str("subject", subject).Msg("echo subscriber started")
    return nil
}

// Stop drains and unsubscribes.
func (s *EchoSubscriber) Stop() error {
    if s.sub != nil {
        return s.sub.Drain()
    }
    return nil
}

func (s *EchoSubscriber) handleEcho(msg *natsclient.Msg) {
    var req model.EchoRequest
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        s.logger.Warn().Err(err).Msg("echo subscriber: malformed payload, ignoring")
        return
    }

    s.logger.Info().
        Str("agent_name", s.agentName).
        Str("community_id", s.communityID).
        Str("tenant_id", req.TenantID).
        Str("message", req.Message).
        Msg("echo request received")

    decorated := model.DecorateMessage(s.agentName, req.Message)
    now := time.Now().UTC()

    reply := model.EchoReply{
        AgentName: s.agentName,
        Decorated: decorated,
        Timestamp: now.Format(time.RFC3339),
    }

    data, err := json.Marshal(reply)
    if err != nil {
        s.logger.Error().Err(err).Msg("echo subscriber: failed to marshal reply")
        return
    }

    if err := msg.Respond(data); err != nil {
        s.logger.Error().Err(err).Msg("echo subscriber: failed to send reply")
    }
}
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm the subscriber does not import any keeper-internal packages other than `domain/model` (the shared value types).
- Confirm `handleEcho` never panics on malformed input — all errors are logged, not propagated.
- Confirm `Stop()` is idempotent (calling twice does not panic).
- Confirm log fields match the zerolog field naming convention used throughout the project (`snake_case`).
