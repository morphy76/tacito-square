# TASK-M8.9-T3: SubscriptionManager Port & NATS Community Subscriber Adapter

| Field       | Value                                          |
|-------------|------------------------------------------------|
| Task ID     | TASK-M8.9-T3                                   |
| Spec        | SPEC-FR-M8.9                                   |
| Boundary    | Application Outbound Port + NATS Inbound Adapter |
| Status      | TODO                                           |
| Depends On  | TASK-M8.9-T2, TASK-M8.9-T4                    |

## Objective

Define the `SubscriptionManager` outbound port and implement `NATSCommunitySubscriber` — the inbound adapter that subscribes to community-scoped NATS subjects and triggers agent scale-up on message arrival.

## Files

| File | Action |
|------|--------|
| `internal/operator/application/ports/outbound/subscription_manager.go` | NEW |
| `internal/operator/adapters/inbound/nats_subscriber.go` | NEW |
| `internal/operator/adapters/inbound/nats_subscriber_test.go` | NEW |

## RED Phase

Create `internal/operator/adapters/inbound/nats_subscriber_test.go` using an in-process NATS server (`nats-server/test` package). Test cases:

- `TestSubscribeSameCommunityOnce`: Calling `Subscribe(ctx, agentA)` and `Subscribe(ctx, agentB)` where both share the same `communityRef` results in exactly **one** subscription in the internal registry, not two.
- `TestSubscribeDifferentCommunities`: Calling `Subscribe` for agents in two different communities results in two distinct subscriptions.
- `TestUnsubscribeLastAgent`: After subscribing one agent and calling `Unsubscribe(ctx, communityRef)`, the subscription is drained and removed from the registry.
- `TestUnsubscribeNotLastAgent`: `Unsubscribe` when other agents still share the same community is a no-op (subscription remains).
- `TestMessageTriggersScaleUp`: Publish a message to `ts.community.{communityRef}.agent.test`; assert `ScaleAgentService.ScaleUpAgent` is called for the mock agent with `spec.replicas==0` and `scaleToZeroEnabled==true`.
- `TestMessageUpdatesHeartbeat`: On message arrival, assert `HeartbeatStore.RecordHeartbeat` is called for all agents in the community.
- `TestMessageSkipsAlreadyRunning`: If the agent already has `spec.replicas >= minReplicas`, `ScaleUpAgent` MUST NOT be called.

Use mock implementations of `ScaleAgentService` and `HeartbeatStore` in tests (hand-rolled mocks, not a mocking framework).

Run `make test` — tests must fail (RED).

## GREEN Phase

**Port interface** — `internal/operator/application/ports/outbound/subscription_manager.go`:

```go
package outbound

import (
    "context"
    "github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
)

// SubscriptionManager is the driven outbound port for managing per-community NATS subscriptions.
type SubscriptionManager interface {
    // Subscribe ensures a NATS subscription exists for the given agent's community.
    // Idempotent: multiple calls for agents in the same community result in a single subscription.
    Subscribe(ctx context.Context, agent *v1alpha1.TacitoAgent) error
    // Unsubscribe drains and removes the community subscription only if no other agents
    // remain in that community. The caller must pass the community reference string.
    Unsubscribe(ctx context.Context, communityRef string) error
}
```

**NATS Subscriber adapter** — `internal/operator/adapters/inbound/nats_subscriber.go`:

Key design points:
- Holds `nc *nats.Conn`, `subs map[string]*nats.Subscription` (keyed by `communityRef`), a `sync.RWMutex`, `k8sClient client.Client`, `scaler inbound.ScaleAgentService`, `heartbeats outbound.HeartbeatStore`, and `logger zerolog.Logger`.
- `Subscribe(ctx, agent)`: acquires write lock; if `communityRef` already in map, return nil (idempotent). Otherwise, call `nc.Subscribe("ts.community.{communityRef}.agent.*", handler)` and store in map.
- `Unsubscribe(ctx, communityRef)`: list all `TacitoAgent` resources with matching `spec.communityRef` in the cluster. If count > 0, return nil. Otherwise drain the subscription and delete from map.
- **Message handler** (runs in the NATS goroutine):
  1. List all `TacitoAgent` with matching `spec.communityRef`.
  2. For each agent: call `heartbeats.RecordHeartbeat(key)`.
  3. For each agent where `scaleToZeroEnabled==true` and `spec.replicas==0`: call `scaler.ScaleUpAgent(ctx, namespace, name)` in a goroutine (non-blocking; log errors).
- Implement compile-time guard: `var _ outbound.SubscriptionManager = (*NATSCommunitySubscriber)(nil)`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Move the communityRef-from-subject parsing (e.g., extracting `{communityRef}` from `ts.community.{communityRef}.agent.*`) into a small, pure, separately-tested helper function.
- Ensure message handler goroutines are bounded by the NATS message context, not leaked.
- Log subscription add/remove at `debug` level; log scale-up triggers at `info` level.
