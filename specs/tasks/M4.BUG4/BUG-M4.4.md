# BUG-M4.4: Redundant Infrastructure Monitoring Services (Prometheus Server & Zipkin Tracing)

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M4.4                                                           |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M4 — Operator Core                                                 |
| Affects       | tools/helm/tacito-square-infra/                                    |
| Violates      | SPEC-NFR-OBSERVABILITY, SPEC-NFR-CLOUD, SPEC-NFR-HPA               |
| Discovered    | Architectural review of observability infrastructure overlaps      |

## Problem Statement

The `tacito-square-infra` Helm chart currently deploys separate, dedicated instances of a Prometheus server and a Zipkin server. This architecture creates several operational issues:
1. **Redundancy & Resource Overlap:** Both systems require stateful persistent storage (Prometheus TSDB, Zipkin storage) and duplicate scraping/forwarding setups.
2. **Protocol Inconsistency:** Distributed tracing spans must be translated by the OpenTelemetry Collector from OTLP format to Zipkin format, adding CPU overhead and translation maintenance.
3. **Heavy Footprint:** Running dedicated metrics and tracing database servers within the local cluster deviates from the stateless, lightweight, cloud-first architecture mandated by `SPEC-NFR-CLOUD`.
4. **Poor Correlation:** Linking metrics (Mimir/Prometheus) and traces (Zipkin) in Grafana is limited due to disconnected backend query APIs and data model differences.

Furthermore, the boundary between the application and infrastructure observability layers is not clearly demarcated:
* **OTel Collector as the Absolute Boundary:** The OTel Collector MUST serve as the strict decoupled boundary. The application components (BFF, Keeper, Agent, Operator) should only have knowledge of the OTel Collector's local OTLP endpoint. They must remain completely oblivious to the downstream telemetry databases (Mimir, Tempo, Grafana, or external SaaS sinks). The infrastructure team must be able to swap the backing storage providers without modifying a single line of application code or redeploying any application pods.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| deploy / helm | [values.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square-infra/values.yaml) | Configures and enables dedicated `prometheus` and `zipkin` local server components. |
| deploy / helm | [Chart.yaml](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square-infra/Chart.yaml) | Includes hardcoded sub-chart dependencies on Prometheus and Zipkin. |
| deploy / helm | [otel-collector/config](file:///Users/R.Pasquini/Projects/side/tacito-square/tools/helm/tacito-square-infra/values.yaml#L156-L188) | Currently pipelines traces to `zipkin` and metrics to a local `prometheus` pull endpoint instead of standard OTLP pushes. |

## Expected Behaviour

### 1. Observability Boundary Enforced
* The application components (Keeper, BFF, Agent, Operator) MUST transmit all metrics and traces via OTLP to the `otel-collector` service at `ts-infra-otel-collector:4317` (gRPC) or `:4318` (HTTP).
* No application component is allowed to possess configurations or environment variables pointing directly to Mimir, Tempo, or Grafana. The OTel Collector acts as the strict interface layer between application and infrastructure.

### 2. Deletion of Prometheus & Zipkin Local Servers
* The `prometheus` and `zipkin` sub-charts MUST be disabled or completely removed from the infrastructure Helm chart.

### 3. OpenTelemetry Native Storage Backends (Tempo & Mimir)
* **Grafana Tempo (Traces Sink):** Introduce a Tempo deployment inside the infrastructure stack as the native OTLP trace storage backend. The OTel Collector forwards traces directly to Tempo using the standard `otlp` gRPC exporter.
* **Grafana Mimir (Metrics Sink):** Configure a Mimir deployment (or a compatible OTLP/Remote Write metric store) in the infrastructure stack. The OTel Collector forwards metrics directly using either the `otlp` or `prometheusremotewrite` exporter.

### 4. Unified Dashboard Layer (Grafana)
* **Grafana Integration:** Grafana is configured to query Mimir for metrics (via PromQL) and Tempo for traces.
* **Trace-Metric exemplars:** Grafana panels query Mimir and use exemplars to link metrics directly to Tempo trace graphs using the shared `trace_id`.

## Licensing & Cost Context (AGPLv3 Compliance)
> [!NOTE]
> Grafana, Mimir, and Tempo are fully open-source/source-available under the **AGPLv3** (GNU Affero General Public License) and are 100% free to self-host, run, and scale for personal or non-commercial projects. Alternatively, Grafana Labs offers a generous **free tier on Grafana Cloud** (up to 3 users, 10K metrics series, 50GB logs, 50GB traces with 14-day retention) which can be utilized by pointing the OTel Collector's outbound OTLP exporters directly to Grafana Cloud endpoints.

## Acceptance Criteria

1. Sub-charts `prometheus` and `zipkin` are disabled or completely removed from `Chart.yaml` and `values.yaml` in the `tacito-square-infra` chart.
2. The `otel-collector` pipelines are updated to push traces via the `otlp` exporter directly to the Tempo endpoint (`ts-infra-tempo:4317` or configured endpoint).
3. The `otel-collector` pipelines are updated to push metrics via the `otlp` (or `prometheusremotewrite`) exporter directly to the Mimir endpoint.
4. Grafana is provisioned and configured with data sources pointing to Mimir and Tempo.
5. All local application services successfully transmit traces and metrics to the OTel Collector, which successfully dispatches them to their respective OTLP sinks.

## Work Items

1. **RED Phase (Validation & Preparation)**:
   - Establish current lint and template baseline by running `make helm-infra-lint` and `make helm-infra-template`.
   - Confirm the presence of `zipkin` and `prometheus` sub-charts in the original `Chart.yaml`.

2. **GREEN Phase (Telemetry Stack Migration)**:
   - Remove `zipkin` and `prometheus` from `Chart.yaml`.
   - Update `values.yaml` to disable/remove references to `zipkin` and `prometheus` and add sections for `tempo`, `mimir`, and `grafana`.
   - Update the `otel-collector` pipeline config in `values.yaml` to route traces to Tempo using the standard `otlp` gRPC exporter and metrics to Mimir using `prometheusremotewrite`.
   - Implement `tempo.yaml`, `mimir.yaml`, and `grafana.yaml` inside `tools/helm/tacito-square-infra/templates/` with monolithic, single-binary configurations and proper services.
   - Run `make helm-infra-deps` to fetch any remaining dependencies and update `Chart.lock`.
   - Run `make helm-infra-lint` and `make helm-infra-template` to verify the refactored chart validates and renders without error.

3. **REFACTOR Phase (Optimization & Standards Compliance)**:
   - Refactor labels, resource limits, and environment variable bindings in `tempo.yaml`, `mimir.yaml`, and `grafana.yaml` templates to conform to the standard `tacito-square-infra` conventions.
   - Verify zero references to Broadcom/Bitnami in the newly added configuration blocks or files.

