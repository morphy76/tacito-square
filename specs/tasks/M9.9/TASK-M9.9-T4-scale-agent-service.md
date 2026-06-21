# TASK-M9.9-T4: Scale Agent Application Service

| Field       | Value                                                |
|-------------|------------------------------------------------------|
| Task ID     | TASK-M9.9-T4                                         |
| Spec        | SPEC-FR-M9.9                                         |
| Boundary    | Application Inbound Port + Application Service Layer |
| Status      | TODO                                                 |
| Depends On  | TASK-M9.9-T1, TASK-M9.9-T2                          |

## Objective

Define the `ScaleAgentService` inbound port and extend `ReconcileAgentServiceImpl` with:
- `ScaleUpAgent`: NATS-triggered scale-up with conflict-safe retry.
- Idle detection inside `Reconcile`: scale to zero when heartbeat is stale.
- Heartbeat recording inside `Reconcile`: seed in-memory store from pod readiness.
- Inject `HeartbeatStore` and `SubscriptionManager` dependencies via constructor.

## Files

| File | Action |
|------|--------|
| `internal/operator/application/ports/inbound/scale_agent_service.go` | NEW |
| `internal/operator/application/service/reconcile_service.go` | MODIFY |
| `internal/operator/application/service/reconcile_service_test.go` | MODIFY |

## RED Phase

Extend `internal/operator/application/service/reconcile_service_test.go` with:

- `TestScaleUpAgentNoop`: `ScaleUpAgent` on an agent with `spec.replicas >= minReplicas` makes no k8s update call.
- `TestScaleUpAgentPatches`: `ScaleUpAgent` on an agent with `spec.replicas==0` patches `spec.replicas = max(minReplicas, 1)`.
- `TestScaleUpAgentConflictRetry`: Simulate a resource-version conflict on the first `Update` call; assert the method retries and succeeds on the second call.
- `TestReconcileIdleScaleDown`: `Reconcile` with `scaleToZeroEnabled=true`, a heartbeat timestamp older than `idleTimeoutSeconds`, and `spec.replicas=1` results in `spec.replicas` being patched to 0.
- `TestReconcileNotIdleWhenDisabled`: `Reconcile` with `scaleToZeroEnabled=false` never modifies `spec.replicas` for idle reasons, regardless of heartbeat age.
- `TestReconcileAlreadyIdle`: `Reconcile` with `spec.replicas==0` and `scaleToZeroEnabled=true` does not re-patch replicas (no-op).
- `TestReconcileRecordsHeartbeatWhenReady`: `Reconcile` where `existingDep.Status.ReadyReplicas > 0` calls `HeartbeatStore.RecordHeartbeat(key)`.
- `TestReconcileNoHeartbeatWhenNotReady`: `Reconcile` where `readyReplicas == 0` does NOT call `HeartbeatStore.RecordHeartbeat`.

Use hand-rolled mock `HeartbeatStore` and `SubscriptionManager` in tests.

Run `make test` — tests must fail (RED).

## GREEN Phase

**New inbound port** — `internal/operator/application/ports/inbound/scale_agent_service.go`:

```go
package inbound

import "context"

// ScaleAgentService defines the driving port for NATS-triggered agent scale-up.
type ScaleAgentService interface {
    // ScaleUpAgent sets spec.replicas to max(minReplicas, 1) for the given agent.
    // It is idempotent: if replicas are already at or above minReplicas, it returns nil.
    ScaleUpAgent(ctx context.Context, namespace, name string) error
}
```

**Extend `ReconcileAgentServiceImpl`**:

1. Add `heartbeats outbound.HeartbeatStore` and `subscriptions outbound.SubscriptionManager` fields.
2. Update `NewReconcileAgentService` constructor to accept both new dependencies.
3. Implement `ScaleUpAgent`:
   - Fetch `TacitoAgent` by `namespace/name`.
   - Determine target replicas: `max(minReplicas, 1)` (use 1 if `MinReplicas` is nil).
   - If `spec.replicas >= target`, return nil.
   - Use `retry.RetryOnConflict(retry.DefaultRetry, func() error { ... client.Update ... })`.
   - Log at `info` with `tenant_id`, `namespace`, `name`.
4. In `Reconcile`, after step 3 (status check), add:
   - **Heartbeat recording**: if `readyReplicas > 0`, call `s.heartbeats.RecordHeartbeat(fmt.Sprintf("%s/%s", agent.Namespace, agent.Name))`.
   - **Idle detection**: if `scaleToZeroEnabled==true` and `spec.replicas > 0`:
     - `lastSeen := s.heartbeats.LastHeartbeat(key)`
     - `timeout := time.Duration(*agent.Spec.IdleTimeoutSeconds) * time.Second` (default 300s if nil)
     - If `!lastSeen.IsZero() && time.Since(lastSeen) >= timeout`: use `retry.RetryOnConflict` to set `spec.replicas = 0`.
5. Add compile-time guard: `var _ inbound.ScaleAgentService = (*ReconcileAgentServiceImpl)(nil)`.

Add import: `"k8s.io/client-go/util/retry"`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Extract `agentKey(agent)` helper returning `"namespace/name"` string — used in both heartbeat recording and idle detection.
- Extract `resolveIdleTimeout(spec)` helper returning `time.Duration` with nil-safe default.
- Extract `resolveMinReplicas(spec)` helper returning `int32` with nil-safe default of 1.
- Ensure all idle-detection log lines include `idle_timeout_seconds` and `last_heartbeat` fields.
