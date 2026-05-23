# BUG-M3.6: Synchronous Blocking Side-Effects in Agent-Community Assignment

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.6                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/adapters/http/assignment_handlers.go              |
| Violates      | SPEC-NFR-REACTIVE §1, §3, §4, §5                                   |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

Under the reactive programming specifications defined in `SPEC-NFR-REACTIVE`, the system must prioritize non-blocking, asynchronous execution and event-driven communication over purely synchronous, imperative workflows. 

However, in the current M3 candidate implementation, the side-effects of **Agent-Community Assignment** (which includes submitting or tearing down Kubernetes CRDs via the operator) are executed completely synchronously inside the HTTP handler thread, blocking the client request:

```go
	// Trigger CRD submission coordinator hook
	if agent, err := h.agentRepo.GetByID(ctx, agentID); err == nil {
		_ = h.crdCoordinator.SubmitAgentCRD(ctx, agent)
	}
```

Although the `crdCoordinator` is currently a no-op stub (`noOpCRDCoordinator`), once it is replaced with the active Kubernetes API coordinator (Milestone M4), any call to `SubmitAgentCRD` or `TeardownAgentCRD` will involve expensive, synchronous network round-trips to the K8s API server. Wrapping this logic directly inside the REST thread violates reactive execution principles.

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `assignment_handlers.go` | `internal/keeper/adapters/http/assignment_handlers.go` | Synchronous invocation of K8s CRD coordinator hooks within HTTP request thread. |

## Impact

1. **Poor API Performance**: Downstream API clients or reverse proxies calling `POST /api/v1/communities/:id/agents/:id` will suffer from high latency and potential timeouts due to blocking K8s API controller calls.
2. **Cascading Failures**: A degraded K8s control plane or operator will cause thread exhaustion in Keeper because the HTTP threads will remain blocked waiting for the CRD hooks to return.
3. **Compliance Failure**: Violates the core requirements of `SPEC-NFR-REACTIVE`.

## Expected Behaviour

1. The HTTP REST layer should only write the assignment change to the repository, write an event, and immediately return a success response to the client.
2. Side-effects (such as submitting/tearing down K8s CRDs) MUST run asynchronously.
3. The system should utilize Go concurrency primitives (channels, goroutines) or an event-driven publish-subscribe model (e.g., local Event Bus or NATS events) to process CRD updates out-of-band.

## Acceptance Criteria

1. API response times for agent assignment remain sub-millisecond and decoupled from the K8s API coordination latency.
2. K8s CRD submissions run in background worker goroutines or react to asynchronously dispatched event streams.
3. Proper context validation is enforced, preventing goroutine leaks when HTTP parent requests cancel or time out.
