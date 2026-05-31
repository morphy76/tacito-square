# BUG-M5.1: Missing Horizontal Pod Autoscaler in Standalone Agent Chart

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M5.1                                                           |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M5 — Agent Core                                                    |
| Affects       | SPEC-FR-M5.7                                                       |
| Violates      | RULE[k8s-best-practices.md] §1 (Horizontal Pod Autoscaling)         |
| Discovered    | Pre-review compliance check of SPEC-FR-M5.7                        |

## Problem Statement

The standalone Agent deployment Helm Chart (`tools/helm/tacito-agent`) does not define a HorizontalPodAutoscaler template (`templates/hpa.yaml`) or provide standard configuration properties for autoscaling under its `values.yaml`.

This violates `RULE[k8s-best-practices.md] §1`, which mandates load-factor-based autoscaling configurations for every deployable microservice (including the `agent` component) with specific scaling thresholds:
- Primary metric: `active_threads` (custom Prometheus metric)
- Secondary metric: CPU utilization
- Limit constraints: Min 1, Max 10 (respecting the community `max_agents` ceiling quota)

## Affected Component and Files

- `[NEW] tools/helm/tacito-agent/templates/hpa.yaml` — missing HorizontalPodAutoscaler definition.
- `[NEW] tools/helm/tacito-agent/values.yaml` — missing scaling structure (`autoscaling.enabled`, `minReplicas`, `maxReplicas`, etc.).

## Impact

1. **Autoscaling standard violation**: The deployment does not adhere to `RULE[k8s-best-practices.md]`.
2. **Resource vulnerability**: Standalone agent pods deployed in high-load production/staging scenarios cannot scale dynamically based on task load/active threads, risking resource exhaustion or message ingestion lag.

## Expected Behaviour

1. The Helm Chart MUST support generating a HorizontalPodAutoscaler resource (`hpa.yaml`) when `autoscaling.enabled` is set to `true`.
2. The HPA template MUST target `autoscaling/v2` and refer to the `apps/v1` `Deployment` of the standalone agent.
3. Standard values for configuring min/max replicas, CPU utilization targets, and custom metric queries (`active_threads`) must be available in `values.yaml`.
4. The template must support exporting `active_threads` custom metric queries mapped through the custom Prometheus adapter.

## Acceptance Criteria

1. Running `helm template` with `--set autoscaling.enabled=true` successfully renders a valid `HorizontalPodAutoscaler` resource.
2. The HPA resource specifies the correct target Deployment and uses `active_threads` (external/object custom metric) alongside standard `cpu` utilization targets.
3. Autoscale replica limits default to min: 1, max: 10, conforming to standard component ceilings.
