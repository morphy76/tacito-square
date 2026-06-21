# SPEC-FR-M9.9: Zero-Scaling Support

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.9                                |
| Status        | ACCEPTED                                    |
| Milestone     | M9                                          |
| Component     | operator                                    |
| Depends On    | SPEC-FR-M4.3                                |
| Supersedes    | none                                        |

## Context

Agents should scale to zero when idle and scale back up when work arrives, conserving cluster resources during inactive periods. For minikube and early-development environments the operator's own control loop — not KEDA or a custom metrics server — is the authoritative mechanism to manage replica counts.

**Scale-to-zero is disabled by default** (`spec.scaleToZeroEnabled: false`) and is explicitly opt-in per agent via the `TacitoAgent` CRD spec.

**Heartbeat tracking is in-memory** for this early development phase. The operator maintains a thread-safe in-process `HeartbeatStore` (a `sync.Map`-backed struct keyed by `{namespace}/{name}`). The existing `status.lastHeartbeat` CRD field is not written to Kubernetes in this phase; it remains reserved for future persistence. The in-memory store is seeded from Kubernetes pod readiness transitions during reconciliation and refreshed on every NATS message arrival.

**Scale-up is driven exclusively by incoming NATS community messages.** When any NATS message arrives on `ts.community.{communityRef}.agent.*`, all scaled-down agents in that community are scaled back up to `minReplicas` (or 1 if unset). The operator subscribes per-community using a wildcard pattern aligned with SPEC-FR-M6.3 (DRAFT).

## Specification

### 1. CRD Schema Extensions (`TacitoAgentSpec`)

The following fields MUST be added to `TacitoAgentSpec` in `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go`:

| Field                | Type      | kubebuilder default | Validation             | Description |
|----------------------|-----------|---------------------|------------------------|-------------|
| `scaleToZeroEnabled` | `*bool`   | `false`             | —                      | Enables scale-to-zero for this agent. Defaults to disabled. |
| `minReplicas`        | `*int32`  | `1`                 | Minimum=0, Maximum=10  | Minimum number of replicas. Used as the scale-up target. |
| `maxReplicas`        | `*int32`  | `1`                 | Minimum=1, Maximum=10  | Maximum number of replicas. |
| `idleTimeoutSeconds` | `*int32`  | `300`               | Minimum=30             | Duration of inactivity in seconds before the agent is scaled to zero. Minimum 30s. |

`zz_generated.deepcopy.go` MUST be regenerated via `make generate` after schema changes.

### 2. In-Memory Heartbeat Store (Outbound Driven Port)

A new outbound port interface `HeartbeatStore` MUST be defined at `internal/operator/application/ports/outbound/heartbeat_store.go`:

```go
type HeartbeatStore interface {
    RecordHeartbeat(key string)          // Records the current time for the given agent key.
    LastHeartbeat(key string) time.Time  // Returns the last recorded time; zero value if never seen.
    Delete(key string)                   // Removes the entry when an agent is deleted.
}
```

The in-memory implementation (`MemoryHeartbeatStore`) MUST be placed at `internal/operator/adapters/outbound/memory/heartbeat_store.go`. It MUST use `sync.Map` internally to be goroutine-safe.

### 3. NATS Connectivity in the Operator

The operator MUST establish a NATS connection on startup using the existing `TS_OPERATOR_NATS_URL` environment variable (bound via Viper key `nats.url`; default: `nats://tacito-infra-nats:4222`).

The NATS connection MUST be:
- Managed with automatic reconnect (`nats.MaxReconnects(-1)`, `nats.ReconnectWait(2*time.Second)`).
- Registered with the shutdown manager to be gracefully drained and closed (`nc.Drain()`).
- Added as a NATS ping check to the `/readyz` probe alongside the existing Kubernetes API check.

### 4. NATS Community Subscriber Inbound Adapter

A new inbound adapter `NATSCommunitySubscriber` MUST be implemented at `internal/operator/adapters/inbound/nats_subscriber.go`. It MUST:

1. **Subscribe per community**: Manage a registry of active `nats.Subscription` instances keyed by community reference (`communityRef`). Multiple agents sharing the same `communityRef` MUST share a single subscription — not one per agent.
2. **Subject pattern**: Subscribe to `ts.community.{communityRef}.agent.*` (aligned with SPEC-FR-M6.3 wildcard pattern).
3. **Implement `SubscriptionManager` outbound port**: Expose `Subscribe(ctx, agent)` and `Unsubscribe(ctx, communityRef)` so the reconciliation application service can manage subscriptions declaratively.
4. **On message arrival**:
   a. Update `HeartbeatStore` for **all** agents belonging to `{communityRef}` (determined by listing `TacitoAgent` resources filtered by `spec.communityRef`).
   b. For every agent in that community where `spec.scaleToZeroEnabled == true` and `spec.replicas == 0`, call `ScaleUpAgent(ctx, namespace, name)` on the `ScaleAgentService` inbound port.
5. **On unsubscribe**: Drain and close the underlying `nats.Subscription` and remove from the registry.

A new outbound port interface `SubscriptionManager` MUST be defined at `internal/operator/application/ports/outbound/subscription_manager.go`:

```go
type SubscriptionManager interface {
    Subscribe(ctx context.Context, agent *v1alpha1.TacitoAgent) error
    Unsubscribe(ctx context.Context, communityRef string) error
}
```

### 5. New Application Inbound Port: `ScaleAgentService`

A new inbound port interface MUST be added at `internal/operator/application/ports/inbound/scale_agent_service.go`:

```go
type ScaleAgentService interface {
    ScaleUpAgent(ctx context.Context, namespace, name string) error
}
```

This interface is implemented by `ReconcileAgentServiceImpl` and called by `NATSCommunitySubscriber` on message arrival.

### 6. Scale-Up Logic (`ScaleUpAgent`)

`ReconcileAgentServiceImpl.ScaleUpAgent` MUST:

1. Fetch the `TacitoAgent` resource by `namespace/name`.
2. If `spec.replicas` is already `>= minReplicas` (or 1 if unset), return early (no-op).
3. Set `spec.replicas = max(minReplicas, 1)`.
4. Persist via `client.Update` wrapped in `retry.RetryOnConflict(retry.DefaultRetry, ...)` to handle concurrent writes safely.
5. Log the scale-up event at `info` level with `tenant_id`, `namespace`, `name`, `communityRef`.

### 7. Scale-to-Zero Control Loop (Idle Detection via Requeue)

The reconciler MUST be extended to drive periodic idle checks:

1. After every successful reconciliation where `spec.scaleToZeroEnabled == true`, the reconciler MUST return `ctrl.Result{RequeueAfter: idleCheckInterval}` where `idleCheckInterval = idleTimeoutSeconds / 2` (minimum 15 seconds).
2. When the reconciler re-fires, it calls `ReconcileAgentService.Reconcile(ctx, agent)` which MUST now include idle detection:
   a. Query `HeartbeatStore.LastHeartbeat(key)`.
   b. If `time.Since(lastHeartbeat) >= idleTimeoutSeconds` AND `spec.replicas > 0`, set `spec.replicas = 0` using `retry.RetryOnConflict`.
   c. The reconciler MUST NOT requeue further once the agent is already at 0 replicas (until a NATS message triggers scale-up and a reconciliation fires again).

### 8. Heartbeat Recording during Reconciliation

`ReconcileAgentServiceImpl.Reconcile` MUST call `HeartbeatStore.RecordHeartbeat(key)` whenever the backing Deployment reports `readyReplicas > 0`. This seeds the in-memory timestamp from pod readiness so that the idle clock begins only after the agent is confirmed healthy.

### 9. Subscription Lifecycle in the Reconciler

The reconciler `Reconcile` method MUST:

1. On agent creation or update: call `SubscriptionManager.Subscribe(ctx, agent)` (idempotent — shared per community).
2. On agent deletion (detected via `DeletionTimestamp`): call `SubscriptionManager.Unsubscribe(ctx, agent.Spec.CommunityRef)` only if no other `TacitoAgent` in the same namespace and community remains. Also call `HeartbeatStore.Delete(key)`.

### 10. Prometheus Metric Fix: `tacito_operator_active_agents` Phase Partitioning

The existing `tacito_operator_active_agents` `Gauge` in `reconciler.go` MUST be replaced with a `GaugeVec` partitioned by the `phase` label (`Pending`, `Running`, `Idle`, `Terminated`):

```go
activeAgents = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "tacito_operator_active_agents",
        Help: "Current number of active TacitoAgent resources by phase",
    },
    []string{"phase"},
)
```

`updateActiveAgentsMetric` MUST list all `TacitoAgent` resources and group counts by `status.phase`, setting each `phase` label bucket independently, resetting all buckets to 0 before recomputing.

### 11. Scale-Up Latency Target

Scale-up latency (time from NATS message arrival to first pod `Ready`) SHOULD be under 30 seconds in minikube environments. This is a soft target for the early development phase.

## Acceptance Criteria

1. **Scale-to-zero disabled by default**: Creating a `TacitoAgent` without `spec.scaleToZeroEnabled: true` MUST NOT cause the operator to scale the agent to 0 replicas, regardless of idle time.
2. **Idle scale-down**: A `TacitoAgent` with `scaleToZeroEnabled: true` and no NATS activity for longer than `idleTimeoutSeconds` MUST have `spec.replicas` set to 0 by the operator, and `status.phase` MUST reflect `Idle`.
3. **NATS-triggered scale-up**: Publishing any NATS message to `ts.community.{communityRef}.agent.*` MUST cause all scaled-down agents in that community with `scaleToZeroEnabled: true` to have `spec.replicas` set to `minReplicas` (or 1 if unset) within 30 seconds.
4. **Heartbeat seeding**: After a pod becomes Ready, the in-memory heartbeat for that agent MUST be updated so the idle clock resets.
5. **Shared community subscription**: Two agents sharing the same `communityRef` MUST result in exactly one NATS subscription, not two.
6. **Subscription cleanup**: Deleting all agents in a community MUST result in the community's NATS subscription being closed.
7. **Conflict-safe scale writes**: Concurrent scale-up (NATS event) and scale-down (requeue) writes MUST not produce stale object errors; all writes MUST use `retry.RetryOnConflict`.
8. **Phase-partitioned metric**: `GET /metrics` MUST return `tacito_operator_active_agents{phase="Idle"}`, `tacito_operator_active_agents{phase="Running"}`, etc. as distinct time series.
9. **NATS readiness check**: `/readyz` MUST report NATS as an unhealthy dependency when the NATS connection is unavailable.
10. **`minReplicas`/`maxReplicas` bounds**: CRD validation MUST reject `minReplicas > maxReplicas` and `idleTimeoutSeconds < 30`.

## Test Plan

### Unit Tests

* **`internal/operator/adapters/outbound/memory/heartbeat_store_test.go`** [NEW]:
  * `RecordHeartbeat` updates the timestamp for a new key.
  * `LastHeartbeat` returns zero time for an unknown key.
  * `Delete` removes the entry.
  * Concurrent reads and writes from multiple goroutines do not produce data races (run with `-race`).

* **`internal/operator/adapters/inbound/nats_subscriber_test.go`** [NEW]:
  * `Subscribe` with two agents sharing the same `communityRef` results in a single `nats.Subscription` registered in the internal map.
  * `Unsubscribe` with remaining agents in the community is a no-op.
  * `Unsubscribe` with no remaining agents drains and removes the subscription.
  * Message arrival calls `ScaleUpAgent` for each scaled-to-zero agent and calls `HeartbeatStore.RecordHeartbeat`.
  * Use `natsserver/test` (`nats-server` in-process) for NATS-level testing.

* **`internal/operator/application/service/reconcile_service_test.go`** [MODIFY]:
  * `ScaleUpAgent` returns early when `spec.replicas >= minReplicas`.
  * `ScaleUpAgent` patches `spec.replicas` to `minReplicas` when agent is at 0.
  * `Reconcile` with `scaleToZeroEnabled: true` and stale heartbeat sets `spec.replicas = 0`.
  * `Reconcile` with `scaleToZeroEnabled: false` never modifies `spec.replicas` for idle detection.
  * `Reconcile` records heartbeat in `HeartbeatStore` when `readyReplicas > 0`.

* **`internal/operator/adapters/inbound/reconciler_test.go`** [MODIFY]:
  * Successful reconcile with `scaleToZeroEnabled: true` returns `RequeueAfter = idleTimeoutSeconds/2`.
  * Successful reconcile with `scaleToZeroEnabled: false` returns `ctrl.Result{}` (no requeue).
  * `updateActiveAgentsMetric` sets the correct counts for each `phase` label.

### Integration Tests (`make test-operator`)

* **Envtest suite** — `internal/operator/adapters/inbound/reconciler_test.go` [MODIFY]:
  * Creating a `TacitoAgent` with `scaleToZeroEnabled: true` and injecting a stale heartbeat (via the `HeartbeatStore` test double) results in the operator patching `spec.replicas = 0` and `status.phase = Idle`.
  * Simulate NATS message via the in-process NATS server; verify `spec.replicas` is set to `minReplicas` for all agents in the community.

### Manual Verification

* Deploy operator to minikube; apply two `TacitoAgent` CRs with the same `communityRef` and `scaleToZeroEnabled: true` and `idleTimeoutSeconds: 60`.
* Wait for idle timeout; confirm both deployments scale to 0 and `kubectl get tacitoagents` shows `Idle`.
* Publish a NATS message with `nats pub ts.community.{communityRef}.agent.test "hello"`; confirm both deployments scale back to 1 and phase returns to `Running`.
* Confirm `curl localhost:8082/metrics` shows `tacito_operator_active_agents{phase="Idle"} 2` and then `{phase="Running"} 2`.

## Files Affected

### CRD Schema
* `pkg/kubernetes/apis/tacito/v1alpha1/tacitoagent_types.go` [MODIFY] — Add `ScaleToZeroEnabled`, `MinReplicas`, `MaxReplicas`, `IdleTimeoutSeconds` fields to `TacitoAgentSpec`.
* `pkg/kubernetes/apis/tacito/v1alpha1/zz_generated.deepcopy.go` [MODIFY] — Regenerated via `make generate`.

### Application Ports
* `internal/operator/application/ports/inbound/scale_agent_service.go` [NEW] — `ScaleAgentService` inbound port interface.
* `internal/operator/application/ports/outbound/heartbeat_store.go` [NEW] — `HeartbeatStore` driven port interface.
* `internal/operator/application/ports/outbound/subscription_manager.go` [NEW] — `SubscriptionManager` driven port interface.

### Adapters
* `internal/operator/adapters/outbound/memory/heartbeat_store.go` [NEW] — `MemoryHeartbeatStore` backed by `sync.Map`.
* `internal/operator/adapters/outbound/memory/heartbeat_store_test.go` [NEW] — Unit tests.
* `internal/operator/adapters/inbound/nats_subscriber.go` [NEW] — `NATSCommunitySubscriber` inbound adapter; also implements `SubscriptionManager`.
* `internal/operator/adapters/inbound/nats_subscriber_test.go` [NEW] — Unit tests with in-process NATS server.
* `internal/operator/adapters/inbound/reconciler.go` [MODIFY] — Fix `activeAgents` to `GaugeVec`, add `RequeueAfter` for idle check, call `SubscriptionManager` on agent lifecycle events.
* `internal/operator/adapters/inbound/reconciler_test.go` [MODIFY] — Add tests for requeue behavior and metric phase partitioning.

### Application Service
* `internal/operator/application/service/reconcile_service.go` [MODIFY] — Add `ScaleUpAgent`, idle detection check, heartbeat recording, `SubscriptionManager` and `HeartbeatStore` dependencies injected via constructor. Implement `ScaleAgentService` port.
* `internal/operator/application/service/reconcile_service_test.go` [MODIFY] — Add tests for new behaviors with mock ports.

### Bootstrap & Wiring
* `internal/operator/bootstrap.go` [MODIFY] — Add NATS readiness checker alongside the existing Kubernetes API checker.
* `cmd/operator/main.go` [MODIFY] — Initialize `nats.Conn`, `MemoryHeartbeatStore`, `NATSCommunitySubscriber`; inject into `ReconcileAgentService` and register NATS drain in shutdown manager.
