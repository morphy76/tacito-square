# SPEC-NFR-HPA: Horizontal Pod Autoscaling

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-HPA                       |
| Status        | DRAFT                              |
| Milestone     | M7                                 |
| Component     | operator, deploy                   |

## Specification

1. Each deployable artifact MUST have an HPA configuration with load-factor-based scaling.
2. Scaling factors per component:

| Component | Primary Metric | Secondary Metric | Min | Max (default) |
|-----------|---------------|-----------------|-----|---------------|
| Agent | `active_threads` (custom) | CPU utilization | 1 | 10 |
| Keeper | HTTP request rate | CPU utilization | 2 | 5 |
| BFF | HTTP request rate | CPU utilization | 2 | 5 |

3. Custom metrics (e.g., `active_threads`) MUST be exported via Prometheus and consumed by the HPA via `prometheus-adapter` or KEDA.
4. HPA templates MUST be part of the Helm chart with configurable min/max/metrics.
5. The Operator MUST set HPA on Agent Deployments it creates, using the community's `max_agents` as the HPA `maxReplicas` ceiling.
6. Scale-to-zero SHOULD be supported for idle agents (configurable, default: disabled).

## Acceptance Criteria

1. HPA Helm template present for keeper, bff
2. Operator creates HPA for agent deployments
3. HPA reacts to `active_threads` custom metric
4. `maxReplicas` respects community `max_agents` quota
5. HPA min/max configurable via values.yaml
