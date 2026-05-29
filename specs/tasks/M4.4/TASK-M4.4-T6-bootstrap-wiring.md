# TASK-M4.4-T6: Bootstrap & Dependency Wiring

| Field       | Value                                           |
|-------------|-------------------------------------------------|
| Task ID     | TASK-M4.4-T6                                    |
| Spec        | SPEC-FR-M4.4                                    |
| Boundary    | Bootstrap / Wiring / Infrastructure Entrypoint  |
| Status      | TODO                                            |
| Depends On  | TASK-M4.4-T3, TASK-M4.4-T4, TASK-M4.4-T5      |

## Objective

Wire all new components into the operator's startup sequence:
- Initialize the NATS connection with auto-reconnect.
- Initialize `MemoryHeartbeatStore`.
- Initialize `NATSCommunitySubscriber`.
- Inject all dependencies into `ReconcileAgentServiceImpl` and `TacitoAgentReconciler`.
- Add NATS ping to the `/readyz` health probe.
- Register NATS `Drain()` in the shutdown manager.

## Files

| File | Action |
|------|--------|
| `internal/operator/bootstrap.go` | MODIFY |
| `cmd/operator/main.go` | MODIFY |

## RED Phase

Extend `internal/operator/bootstrap_test.go`:

- `TestNATSCheckerHealthy`: Create a `NATSChecker` with a connected in-process NATS server; assert `Check(ctx)` returns nil.
- `TestNATSCheckerUnhealthy`: Pass a closed/nil NATS connection; assert `Check(ctx)` returns a non-nil error.

Run `make test` — tests must fail (RED).

## GREEN Phase

**`internal/operator/bootstrap.go`** — add `NATSChecker` factory:

```go
// NATSChecker creates a health.Checker that pings the NATS connection.
func NATSChecker(nc *nats.Conn) health.Checker {
    return health.Checker{
        Name: "nats",
        Check: func(ctx context.Context) error {
            if nc == nil || !nc.IsConnected() {
                return errors.New("nats connection is not established")
            }
            return nil
        },
    }
}
```

Import `"github.com/nats-io/nats.go"`.

**`cmd/operator/main.go`** — extend `main()` after step 5 (k8s manager init):

```go
// 6a. Initialize NATS connection
natsURL := v.GetString("nats.url")
if natsURL == "" {
    natsURL = "nats://tacito-infra-nats:4222"
}
nc, err := nats.Connect(natsURL,
    nats.MaxReconnects(-1),
    nats.ReconnectWait(2*time.Second),
    nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
        logger.Warn().Err(err).Msg("nats disconnected")
    }),
    nats.ReconnectHandler(func(_ *nats.Conn) {
        logger.Info().Msg("nats reconnected")
    }),
)
if err != nil {
    logger.Fatal().Err(err).Str("nats_url", natsURL).Msg("failed to connect to NATS")
}
mgr.Register("nats", func(ctx context.Context) error {
    logger.Info().Msg("draining nats connection")
    return nc.Drain()
})

// 6b. Initialize in-memory HeartbeatStore
heartbeatStore := memory.NewMemoryHeartbeatStore()

// 6c. Initialize application service with all dependencies
reconcileService := service.NewReconcileAgentService(k8sMgr.GetClient(), logger, v, heartbeatStore, /* subscriptionManager injected below */)

// 6d. Initialize NATS subscriber (implements SubscriptionManager outbound port)
natsSubscriber := inbound.NewNATSCommunitySubscriber(nc, k8sMgr.GetClient(), reconcileService, heartbeatStore, logger)

// 6e. Re-wire service with subscriptionManager (or pass to constructor directly)
reconcileService.SetSubscriptionManager(natsSubscriber)

// 6f. Initialize reconciler with all dependencies
reconciler := inbound.NewTacitoAgentReconciler(k8sMgr.GetClient(), scheme, reconcileService, natsSubscriber, heartbeatStore, logger)
```

> **Note on circular dependency**: `ReconcileAgentServiceImpl` needs `SubscriptionManager` and `NATSCommunitySubscriber` needs `ScaleAgentService` (implemented by `ReconcileAgentServiceImpl`). Break the cycle using a `SetSubscriptionManager(outbound.SubscriptionManager)` setter on `ReconcileAgentServiceImpl` that is called after both are constructed — or refactor the constructor to accept an optional setter. Document this wiring order explicitly with a comment in `main.go`.

**Update `/readyz`** — add NATS checker:

```go
kubeAPIChecker := operator.KubeAPIChecker(k8sMgr.GetAPIReader())
natsChecker := operator.NATSChecker(nc)
router := operator.NewServer(kubeAPIChecker, natsChecker)
```

Run `make test` (unit) and `make build-operator` — must compile and pass (GREEN).

## REFACTOR Phase

- Ensure `v.SetDefault("nats.url", "nats://tacito-infra-nats:4222")` is set at the top of `main()` alongside other Viper defaults.
- Remove the circular-dependency setter pattern if a cleaner constructor injection order can be found.
- Confirm graceful shutdown order: HTTP server drains first, then controller manager stops, then NATS drains, then OTel tracer flushes.
- Run `make build-operator` and confirm binary compiles without warnings.
- Run `make test` — all tests GREEN.

## Post-Task Verification

After T6 completes, run the full manual verification from the spec's Test Plan:
1. Deploy operator to minikube with two agents sharing a `communityRef` and `scaleToZeroEnabled: true`, `idleTimeoutSeconds: 60`.
2. Confirm idle scale-down to 0 replicas after 60 seconds.
3. Publish `nats pub ts.community.{communityRef}.agent.test "wake"` and confirm both scale back up.
4. Confirm `curl :8082/metrics` shows `tacito_operator_active_agents{phase="Idle"}` and `{phase="Running"}` as distinct series.
5. Confirm `curl :8082/readyz` lists `nats` as a healthy dependency.
