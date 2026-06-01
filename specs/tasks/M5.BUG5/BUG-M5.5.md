# BUG-M5.5: Agent Component Does Not Export Prometheus Metrics

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.5                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | [bootstrap.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/bootstrap.go), [main.go](file:///Users/R.Pasquini/Projects/side/tacito-square/cmd/agent/main.go) |
| Violates      | SPEC-NFR-OBSERVABILITY, SPEC-NFR-HPA                               |
| Discovered    | Code audit and inspection of agent HTTP server bootstrapping       |

## Problem Statement

The `agent` component does not expose any Prometheus metrics or register the standard Prometheus metrics handler. 

In `internal/agent/bootstrap.go`, `NewServer` only configures `/healthz` and `/readyz` endpoints. The standard Gin metrics middleware (`observability.MetricsMiddleware()`) is not configured, and the standard `/metrics` endpoint wrapping `observability.MetricsHandler()` is missing entirely. 

This violates:
1. **`SPEC-NFR-OBSERVABILITY`**: Requires every deployable component to support standard Prometheus metrics, including request count, latency histograms, error rates, and runtime statistics.
2. **`SPEC-NFR-HPA`**: Requires autoscaling the agent component using active threads and CPU utilization, which depends directly on these metrics being exposed for scraping by Prometheus.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Agent / Bootstrap | [internal/agent/bootstrap.go](file:///Users/R.Pasquini/Projects/side/tacito-square/internal/agent/bootstrap.go) | Missing `MetricsMiddleware()` registration and `r.GET("/metrics")` endpoint routing. |
| Agent / Main | [cmd/agent/main.go](file:///Users/R.Pasquini/Projects/side/tacito-square/cmd/agent/main.go) | Lacks metrics exporter registration during initial component startup. |

## Proposed Agent Metrics Catalog

To support granular observability, capacity planning, and horizontal autoscaling, the agent component will export a tailored set of metrics categorized by architectural boundary:

### Detailed Metrics Schema

| Category | Metric Name | Type | Labels | Description / Unit |
|----------|-------------|------|--------|---------------------|
| **Core / Autoscaling** | `active_threads` | UpDownCounter | `agent`, `community` | Number of concurrent agent reasoning executions (integer gauge-like). Primary metric for HPA. |
| **NATS Messaging** | `agent_nats_messages_processed_total` | Counter | `agent`, `community`, `subject`, `status` | Total NATS messages processed by the agent. `status` is `success` or `error`. |
| | `agent_nats_processing_duration_seconds` | Histogram | `agent`, `community`, `subject` | Time taken to process and reply to a NATS message (seconds). |
| **Short-Term Memory (STM)** | `agent_stm_operations_total` | Counter | `agent`, `operation`, `status` | Total Redis short-term memory interactions. `operation` tracks `read`, `write`, `delete`. |
| | `agent_stm_operation_duration_seconds` | Histogram | `agent`, `operation` | Latency of short-term memory operations (seconds). |
| **Long-Term Memory (LTM)** | `agent_ltm_operations_total` | Counter | `agent`, `operation`, `status` | Total Qdrant LTM interactions. `operation` tracks `search`, `upsert`. |
| | `agent_ltm_operation_duration_seconds` | Histogram | `agent`, `operation` | Latency of long-term memory operations (seconds). |
| **Cognitive Brain (LLM)** | `agent_brain_requests_total` | Counter | `agent`, `provider`, `model`, `status` | Total execution requests dispatched to the LLM backend (OpenAI or Ollama). |
| | `agent_brain_request_duration_seconds` | Histogram | `agent`, `provider`, `model` | Latency of the LLM API reasoning roundtrip (seconds). |
| | `agent_brain_tokens_total` | Counter | `agent`, `direction`, `model` | Count of tokens processed. `direction` tracks `sent` (prompt) or `received` (completion). |

### Keeper vs. Agent Observability Segregation

To maintain clean architectural isolation (DDD / Hexagonal separation), the agent component's local `/metrics` endpoint MUST NOT export metrics that belong strictly to the global multi-tenant orchestration layer (managed by the `keeper` component).

The following metrics defined in the shared observability module are **excluded** from the agent's local metrics scope:
*   `agent_status`: Excluded because an individual agent runtime instance only knows its own state and cannot monitor the status of other agents.
*   `pending_hitl_callbacks`: Excluded because Human-In-The-Loop callback queues are orchestrated globally at the keeper layer.
*   `community_quota_utilization` / `agent_quota_utilization`: Excluded because budget enforcement and token/request quota allocations are verified and tracked centrally by the keeper component.

Instead, the agent component will exclusively reuse or define:
1.  **Shared Technical Metrics**:
    *   `http_requests_total` and `http_request_duration_seconds` (via `observability.MetricsMiddleware()`).
    *   `active_threads` (UpDownCounter) to track active message/cognitive execution loops.
    *   `outbound_dependency_duration_seconds` (Histogram) to capture latencies of external ports (`redis`, `qdrant`, `openai`/`ollama`).
2.  **Isolated Agent Domain Metrics**:
    *   NATS processing throughput/latencies, STM/LTM operations, and brain-specific token counts as mapped in the detailed schema above.

### Rate Scenarios & PromQL Queries

These scenario queries will be used in Grafana dashboards and alerts to evaluate agent operational performance and throughput rates:

#### 1. Message Processing Rate (Throughput)
*   **Scenario**: Monitor how many incoming messages each agent is actively handling per second.
*   **PromQL Query**:
    ```promql
    sum(rate(agent_nats_messages_processed_total[5m])) by (agent)
    ```

#### 2. Cognitive Token Ingestion and Generation Rates
*   **Scenario**: Track prompt and completion token ingestion/generation speeds to optimize token allocations, predict costs, and identify large payload trends.
*   **PromQL Query (Tokens per Second)**:
    ```promql
    sum(rate(agent_brain_tokens_total[5m])) by (agent, direction)
    ```

#### 3. Database & Memory I/O Rates
*   **Scenario**: Detect read/write bottlenecks in short-term memory (Redis) and long-term memory (Qdrant) by tracking database operations per second.
*   **PromQL Queries (Operations per Second)**:
    *   *Short-Term Memory*:
        ```promql
        sum(rate(agent_stm_operations_total[5m])) by (agent, operation)
        ```
    *   *Long-Term Memory*:
        ```promql
        sum(rate(agent_ltm_operations_total[5m])) by (agent, operation)
        ```

#### 4. Execution Success vs. Failure Ratios (Error Rates)
*   **Scenario**: Set up threshold alerts to detect when an agent's failure rate exceeds a specific percentage of overall traffic (e.g., alert if error rate is greater than 5%).
*   **PromQL Query (Failure Rate Percentage)**:
    ```promql
    sum(rate(agent_nats_messages_processed_total{status="error"}[5m])) by (agent) 
    / 
    sum(rate(agent_nats_messages_processed_total[5m])) by (agent) * 100
    ```

#### 5. Brain API Average Latency Profile
*   **Scenario**: Analyze average roundtrip latency of LLM reasoning requests to detect provider slow-downs or high inference times.
*   **PromQL Query (Average Latency in Seconds)**:
    ```promql
    sum(rate(agent_brain_request_duration_seconds_sum[5m])) by (agent, provider, model) 
    / 
    sum(rate(agent_brain_request_duration_seconds_count[5m])) by (agent, provider, model)
    ```

## Impact

1. **Autoscaling Failures**: Since Prometheus metrics are missing, the Horizontal Pod Autoscaler (HPA) cannot retrieve the `active_threads` or execution load metrics, making dynamic scaling impossible.
2. **Observability Blind Spots**: Operational metrics for HTTP request latencies, NATS/Redis outbound dependencies, and system runtime cannot be collected or monitored for the agent component.

## Expected Behaviour

1. The agent component's HTTP Gin server MUST register the `observability.MetricsMiddleware()` to collect HTTP metrics.
2. The agent HTTP server MUST expose a `/metrics` route that maps to `observability.MetricsHandler()`.
3. The `/metrics` endpoint itself MUST be excluded from trace spans and metrics logging to avoid noisy logs and telemetry pollution (similar to the `/healthz` and `/readyz` endpoints).

## Acceptance Criteria

1. Scraping `GET /metrics` on the agent component's HTTP server returns a `200 OK` status with valid Prometheus exposition format metrics.
2. The metrics payload includes standard runtime metrics, HTTP request duration histograms, and dependencies metrics.
3. Automated unit and integration tests are added to verify that `GET /metrics` returns the expected Prometheus payload.
