# TASK-M2.1.1: Application chart validation script (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.1.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.1                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that defines the expected behavior of the application Helm chart after refactoring. Verifies all component templates render correctly, health probes are configured, CRDs are included, and no infrastructure sub-charts remain.

## Work Items

1. Create `test/helm/test_app_chart.sh` with the following checks:
   - `helm lint tools/helm/tacito-square/` passes.
   - `helm template` renders a Deployment for each component: keeper, agent, operator (when enabled), bff (when enabled).
   - `helm template` renders a Service for keeper, bff (when enabled).
   - Each Deployment configures liveness probe at `/healthz` and readiness probe at `/readyz`.
   - Template output contains `TacitoAgent` and `TacitoCommunity` CRD definitions.
   - `Chart.yaml` has zero `dependencies` entries.
   - Operator and BFF are conditionally renderable via `operator.enabled` / `bff.enabled`.
   - Component image tags reference the values from `values.yaml`.
2. Run the script — it MUST FAIL (templates don't exist yet or are incomplete).

## Acceptance Criteria

1. `test/helm/test_app_chart.sh` exists and is executable.
2. Running the script produces clear FAIL output for missing/incomplete templates.
