# BUG-M4.2: Health and Metrics Endpoints Leak Tracing Spans

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M4.2                                                           |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M4 — Operator Core                                                 |
| Affects       | internal/shared/observability/tracing.go                           |
| Violates      | SPEC-FR-M2.2, SPEC-NFR-OBSERVABILITY                               |
| Discovered    | Code inspection of telemetry endpoints during M4 integration        |

## Problem Statement

The OpenTelemetry `TracingMiddleware` implemented in [tracing.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/shared/observability/tracing.go) does not check the request path before starting a span. As a result, every request to health probes (`/healthz`, `/readyz`) and metrics scraping endpoints (`/metrics`) generates and exports a server span. 

In a production environment, Kubernetes frequently scrapes health endpoints (every 5-10 seconds per replica), resulting in a massive volume of redundant, low-value telemetry. This floods the OpenTelemetry Collector and Zipkin/Jaeger backends, degrades system performance, increases telemetry ingestion and storage costs, and pollutes search logs.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| shared / observability | [tracing.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/shared/observability/tracing.go) | `TracingMiddleware` starts a server span unconditionally for all incoming HTTP requests. |

## Impact

1. **Telemetry Storage & Network Flooding**: Heavy trace emission under frequent health scraping schedules, causing high storage/bandwidth overhead on OpenTelemetry backend collectors.
2. **Diagnostic Pollution**: Trace databases become crowded with high volumes of identical, successful health check traces, making it difficult to search for and trace legitimate user/system transactions.

## Expected Behaviour

1. The OpenTelemetry HTTP request middleware `TracingMiddleware` MUST bypass span generation when the request path matches `/healthz`, `/readyz`, or `/metrics`.
2. The middleware must still extract and propagate incoming tracing headers (e.g. `traceparent`) to the request context to preserve trace propagation context down the call stack, but it must not generate a local server span for these endpoints.

## Acceptance Criteria

1. Making a request to `/healthz`, `/readyz`, or `/metrics` does not record or export any OpenTelemetry span.
2. Making a request to normal endpoints (e.g., `/api/v1/echo`) still successfully records and exports spans.
3. Existing unit test suites in `internal/shared/observability/tracing_test.go` remain functional and are updated to assert this path-exclusion behavior.
