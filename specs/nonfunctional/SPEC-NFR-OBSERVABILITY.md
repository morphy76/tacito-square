# SPEC-NFR-OBSERVABILITY: Observability (Metrics, Tracing, and Correlation)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-OBSERVABILITY             |
| Status        | ACCEPTED                           |
| Component     | keeper, agent, bff, operator       |

## Specification

1. **Three Pillars of Observability**: Every deployable artifact (agent, keeper, operator, bff) MUST support three integrated pillars of observability: Metrics, Tracing, and Structured Logging.
2. **Prometheus Metrics**:
   - Every deployable component MUST expose a `/metrics` endpoint in the Prometheus exposition format.
   - Metrics MUST include:
     - **HTTP**: request count, latency histogram, and error rate (by route, method, status)
     - **Runtime**: goroutines, memory allocation, GC pauses
     - **Domain**: agent count (by status), active threads, and pending HITL callbacks
     - **Quota**: utilization gauges per community and agent
     - **Dependency**: outbound call latency histograms (LLM, Redis, Qdrant, NATS, PostgreSQL, S3)
3. **Distributed Tracing (OpenTelemetry)**:
   - All HTTP, NATS, and DB interactions MUST be instrumented with **OpenTelemetry (OTel)** tracing.
   - Trace contexts (W3C traceparent headers) MUST be propagated across all RPC, HTTP, and NATS boundaries (Context Propagation).
   - Spans MUST include standardized attributes such as component name, service version, environment, query execution state, and relevant error statuses.
4. **Observability Correlation**:
   - Traces, logs, and metrics MUST be correlated to enable seamless end-to-end troubleshooting.
   - Every log entry (as specified in `SPEC-NFR-LOG`) MUST inject `trace_id` and `span_id` when active span contexts exist.
   - Inbound HTTP routes and NATS subscribers MUST extract the incoming trace context, start a child span, and inject context before propagating downstream.
5. **Infrastructure**:
   - Prometheus ServiceMonitor CRDs and OpenTelemetry Collector configurations SHOULD be included in Helm templates.

## Acceptance Criteria

1. `GET /metrics` returns Prometheus text format on all runnable components.
2. HTTP request histograms and dependency latency metrics are present and well-labeled.
3. OpenTelemetry tracer is initialized with context propagation enabled on HTTP handlers, NATS subscribers, and DB client adapters.
4. Tracing spans are correctly linked across boundaries (e.g., BFF -> Keeper -> Database / NATS).
5. All JSON logs produced during a request carry standard `trace_id` and `span_id` fields matching the active OTel context.
