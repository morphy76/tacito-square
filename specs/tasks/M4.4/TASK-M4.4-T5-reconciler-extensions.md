# TASK-M4.4-T5: Reconciler Extensions

| Field       | Value                                           |
|-------------|-------------------------------------------------|
| Task ID     | TASK-M4.4-T5                                    |
| Spec        | SPEC-FR-M4.4                                    |
| Boundary    | Inbound Adapter — Kubernetes Reconciler         |
| Status      | TODO                                            |
| Depends On  | TASK-M4.4-T3, TASK-M4.4-T4                     |

## Objective

Extend the existing `TacitoAgentReconciler` with:
1. `RequeueAfter` on every successful reconciliation when `scaleToZeroEnabled=true`.
2. Subscription lifecycle management (subscribe on create/update, unsubscribe on last deletion).
3. Fix `tacito_operator_active_agents` from a plain `Gauge` to a `GaugeVec` partitioned by `phase`.

## Files

| File | Action |
|------|--------|
| `internal/operator/adapters/inbound/reconciler.go` | MODIFY |
| `internal/operator/adapters/inbound/reconciler_test.go` | MODIFY |

## RED Phase

Extend `internal/operator/adapters/inbound/reconciler_test.go`:

- `TestReconcileRequeuesWhenScaleToZeroEnabled`: Mock service returning nil; agent has `scaleToZeroEnabled=true` and `idleTimeoutSeconds=120`. Assert returned `ctrl.Result.RequeueAfter == 60*time.Second` (half of 120).
- `TestReconcileNoRequeueWhenScaleToZeroDisabled`: Agent has `scaleToZeroEnabled=false`. Assert returned `ctrl.Result.RequeueAfter == 0`.
- `TestReconcileNoRequeueWhenAlreadyIdle`: Agent has `scaleToZeroEnabled=true` and `spec.replicas==0`. Assert no requeue (the NATS message will re-trigger).
- `TestReconcileCallsSubscribeOnUpsert`: Reconciler calls `SubscriptionManager.Subscribe(ctx, agent)` for a live agent.
- `TestReconcileCallsUnsubscribeOnDeletion`: Agent has non-nil `DeletionTimestamp` and no other agents in the community; reconciler calls `SubscriptionManager.Unsubscribe(ctx, communityRef)` and `HeartbeatStore.Delete(key)`.
- `TestUpdateActiveAgentsMetricByPhase`: After two Running and one Idle agent, `updateActiveAgentsMetric` sets `phase="Running"` gauge to 2 and `phase="Idle"` gauge to 1.

Run `make test` — tests must fail (RED).

## GREEN Phase

**Metric fix** in `reconciler.go`:

Replace:
```go
activeAgents = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Name: "tacito_operator_active_agents",
        Help: "Current number of active TacitoAgent resources",
    },
)
```

With:
```go
activeAgents = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "tacito_operator_active_agents",
        Help: "Current number of active TacitoAgent resources by phase",
    },
    []string{"phase"},
)
```

Pre-initialize all four phase label series in `init()`:
```go
activeAgents.WithLabelValues(string(v1alpha1.PhasePending))
activeAgents.WithLabelValues(string(v1alpha1.PhaseRunning))
activeAgents.WithLabelValues(string(v1alpha1.PhaseIdle))
activeAgents.WithLabelValues(string(v1alpha1.PhaseTerminated))
```

**`updateActiveAgentsMetric`** — rewrite to group by phase:
```go
func (r *TacitoAgentReconciler) updateActiveAgentsMetric(ctx context.Context) {
    // Reset all phase counters
    for _, phase := range []v1alpha1.TacitoAgentPhase{
        v1alpha1.PhasePending, v1alpha1.PhaseRunning,
        v1alpha1.PhaseIdle, v1alpha1.PhaseTerminated,
    } {
        activeAgents.WithLabelValues(string(phase)).Set(0)
    }
    var list v1alpha1.TacitoAgentList
    if err := r.client.List(ctx, &list); err == nil {
        counts := map[v1alpha1.TacitoAgentPhase]float64{}
        for _, agent := range list.Items {
            counts[agent.Status.Phase]++
        }
        for phase, count := range counts {
            activeAgents.WithLabelValues(string(phase)).Set(count)
        }
    }
}
```

**RequeueAfter logic** — after successful `r.service.Reconcile(ctx, agent)`:
```go
if agent.Spec.ScaleToZeroEnabled != nil && *agent.Spec.ScaleToZeroEnabled &&
    (agent.Spec.Replicas == nil || *agent.Spec.Replicas > 0) {
    timeout := int32(300)
    if agent.Spec.IdleTimeoutSeconds != nil {
        timeout = *agent.Spec.IdleTimeoutSeconds
    }
    requeueAfter := time.Duration(timeout/2) * time.Second
    if requeueAfter < 15*time.Second {
        requeueAfter = 15 * time.Second
    }
    return ctrl.Result{RequeueAfter: requeueAfter}, nil
}
return ctrl.Result{}, nil
```

**Subscription lifecycle** — add `subscriptions outbound.SubscriptionManager` and `heartbeats outbound.HeartbeatStore` fields to `TacitoAgentReconciler`. Inject via `NewTacitoAgentReconciler`. In `Reconcile`:
- If agent has a non-nil `DeletionTimestamp`:
  - List remaining agents with same `spec.communityRef`.
  - If count == 0: call `r.subscriptions.Unsubscribe(ctx, agent.Spec.CommunityRef)` and `r.heartbeats.Delete(key)`.
  - Return early.
- Otherwise: call `r.subscriptions.Subscribe(ctx, agent)`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Extract `idleRequeueAfter(agent)` pure helper for the requeue calculation.
- Confirm `updateActiveAgentsMetric` does not call `List` twice in a single reconcile loop.
- All existing reconciler tests must remain GREEN.
