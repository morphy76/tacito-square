# SPEC-NFR-METRICS: Prometheus Metrics Endpoints

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-METRICS                   |
| Status        | ACCEPTED                           |
| Component     | keeper, agent, bff                 |

## Specification

1. Every deployable artifact (agent, keeper, operator, bff) MUST expose a `/metrics` endpoint in Prometheus exposition format.
2. Metrics MUST include:
   - **HTTP**: request count, latency histogram, error rate (by route, method, status)
   - **Runtime**: goroutines, memory alloc, GC pause
   - **Domain**: agent count (by status), active threads, HITL pending callbacks
   - **Quota**: utilization gauges per community and agent
   - **Dependency**: outbound call latency (LLM, Redis, Qdrant, NATS, PostgreSQL, S3)
3. Gin middleware MUST auto-instrument HTTP metrics.
4. Prometheus ServiceMonitor CRDs SHOULD be included in Helm templates.

## Acceptance Criteria

1. `GET /metrics` returns Prometheus text format on all components
2. HTTP request histogram present with route/method/status labels
3. Domain-specific gauges present (agent_count, active_threads)
4. Dependency call latency histograms present
