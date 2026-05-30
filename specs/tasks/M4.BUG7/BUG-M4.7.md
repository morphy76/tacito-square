# BUG-M4.7: Agent Pod Never Subscribes to NATS Echo Subject — EchoSubscriber Not Wired in Bootstrap

| Field         | Value                                                              |
|---------------|-------------------------------------------------------------------|
| ID            | BUG-M4.7                                                          |
| Status        | CLOSED                                                       |
| Severity      | HIGH                                                              |
| Milestone     | M4 — Operator Core                                                |
| Affects       | cmd/agent/main.go                                                 |
| Violates      | SPEC-FR-M4.8                                                      |
| Discovered    | Runtime testing of the community echo endpoint                    |

## Problem Statement

After fixing the NATS subject mismatch (BUG-M4.6), the community echo endpoint still fails with:

```
"nats request to ts.community.0c8d02af-6235-4fae-825d-694c8547d09c.agent.qa-agent: nats: no responders available for request"
```

Investigation shows that although `EchoSubscriber` is correctly implemented in
`internal/agent/adapters/inbound/nats/echo_subscriber.go`, **the agent's `cmd/agent/main.go`
never connects to NATS and never instantiates or starts the `EchoSubscriber`**.

The agent pod starts an HTTP health server and then enters a blocking wait for OS signals,
but never registers any NATS subscription. As a result NATS has no responder for the echo
subject, regardless of whether the pod is `Running` or how many replicas are ready.

The Kubernetes Operator correctly injects the following env vars into every agent pod:
- `TS_AGENT_NATS_URL`
- `TS_AGENT_NAME`
- `TS_AGENT_COMMUNITY_REF` (UUID of the assigned community)
- `TS_AGENT_LOG_LEVEL`

These env vars are mapped by the `config.Load("TS_AGENT")` Viper configuration, so the values
are already available inside the agent process as `nats.url`, `name`, `community.ref`, and
`log.level` — they just need to be read and used.

The T6 task (TASK-M4.8-T6) that implemented `EchoSubscriber` explicitly deferred agent
bootstrap wiring to M5 (scope note: "Full agent wiring is deferred to M5"). However, without
this wiring the entire echo capability is non-functional at runtime, constituting a bug in the
M4 milestone.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| agent / entrypoint | [main.go](file:///Users/R.Pasquini/Projects/side/tacito-square/cmd/agent/main.go) | Never reads `nats.url`, never connects to NATS, never starts `EchoSubscriber`. |

## Impact

1. **Echo Endpoint Fully Broken at Runtime:** Any call to `/communities/:community_id/echo` that
   targets a running agent will receive a NATS "no responders" error.
2. **Misleading Pod Status:** The agent pod appears `Running`/`Ready` via Kubernetes health probes
   but is functionally dead with respect to all NATS-based messaging.

## Expected Behaviour

When the agent pod starts:
1. It reads `TS_AGENT_NATS_URL`, `TS_AGENT_NAME`, and `TS_AGENT_COMMUNITY_REF` from the
   environment (via Viper with prefix `TS_AGENT`).
2. It establishes a NATS connection to `nats.url`.
3. It instantiates and calls `Start()` on an `EchoSubscriber` configured with `name`,
   `community.ref`, and the tenant ID (empty string acceptable for M4 — pre-auth milestone).
4. The `EchoSubscriber.Stop()` is registered with the shutdown manager for graceful drain.

## Acceptance Criteria

1. After deploying the fixed agent image, a call to `/communities/:community_id/echo` with a
   running agent returns a successful `200 OK` with decorated agent replies.
2. The agent pod logs `"echo subscriber started"` with the correct subject on startup.
3. If `TS_AGENT_NATS_URL` is not set, the agent logs a fatal error and exits — it cannot operate
   without NATS connectivity.
