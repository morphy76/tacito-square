# BUG-M3.4: Broken Observability Context Propagation & Domain Metric Gaps

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.4                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/bootstrap.go, internal/shared/observability/metrics.go |
| Violates      | SPEC-NFR-OBSERVABILITY §2, §3, §4                                  |
| Discovered    | M3 candidate post-implementation NFR review                         |

## Problem Statement

The unified observability framework in the current Milestone M3 candidate implementation contains major tracking and integration gaps:

1. **Broken Distributed Tracing Context Propagation**: 
   Inside `internal/keeper/bootstrap.go`, the Gin HTTP router is initialized with logging, recovery, and metrics middlewares but **completely lacks OTel request tracing middleware** (e.g., `otelgin` or manual header extraction). 
   Although OTel tracing is initialized and spans are manually started in handlers using `c.Request.Context()`:
   ```go
   ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.create_agent", ...)
   ```
   since the incoming request context is never populated with extracted W3C traceparent headers from incoming HTTP requests, every inbound request creates a brand new root span. This breaks end-to-end trace correlation and context propagation from upstream components (like BFF or external API clients).
2. **Prometheus Domain, Quota, and Dependency Metric Gaps**:
   The `/metrics` endpoint is live but only exposes HTTP duration/totals and basic pgx database pool stats. It is missing the mandatory domain-specific gauges and counters outlined in `SPEC-NFR-OBSERVABILITY`:
   - **Domain metrics**: agent counts (by status), active threads, pending HITL callbacks.
   - **Quota metrics**: utilization gauges per community and agent.
   - **Dependency latency metrics**: outbound call latency histograms (Qdrant, Redis, NATS, S3).

## Affected Aggregates and Files

| File / Component | Location | Issue |
|------------------|----------|-------|
| `bootstrap.go` | `internal/keeper/bootstrap.go` | Missing OTel trace-extraction middleware; context propagation is broken. |
| `metrics.go` | `internal/shared/observability/metrics.go` | Missing domain, quota, and dependency latency collectors. |

## Impact

1. **Broken Distributed Tracing**: Operations spanning multiple services (e.g., BFF calling Keeper, which then reconciles via the Operator) cannot be traced as a single correlated request tree.
2. **Missing Monitoring Visibility**: SREs and Operators cannot monitor agent system health (e.g., community quotas, active thread counts) because metrics are absent from `/metrics`.
3. **Compliance Failure**: Fails to satisfy SPEC-NFR-OBSERVABILITY AC-2, AC-3, and AC-4.

## Expected Behaviour

1. The HTTP server MUST register an OTel context-extraction middleware that parses incoming W3C traceparent headers and stores the trace context inside Go's `context.Context` for every HTTP route.
2. Manually started spans inside HTTP handlers must successfully link to the incoming parent trace.
3. The Prometheus registry MUST declare and expose:
   - `active_threads` and `agent_status` gauges.
   - Quota utilization gauges.
   - Outbound dependency latency histograms (Redis, NATS, etc.).

## Acceptance Criteria

1. Verification using trace headers (e.g. `traceparent`) in test calls shows spans successfully link to the parent context.
2. Accessing `/metrics` exposes valid metrics for `active_threads`, `agent_status`, community/agent quota gauges, and dependency latencies.
3. Prometheus ServiceMonitor targets successfully scrape the enriched variables.
