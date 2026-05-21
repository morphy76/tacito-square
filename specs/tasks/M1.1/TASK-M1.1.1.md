# TASK-M1.1.1: Infrastructure chart validation script (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.1.1                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.1                                |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that defines the expected behavior of the infrastructure Helm chart. This script will fail initially because the chart does not exist yet — establishing the RED phase of TDD.

## Work Items

1. Create `test/helm/test_infra_chart.sh` with the following checks:
   - `helm lint tools/helm/tacito-square-infra/` passes.
   - `helm template tacito-infra tools/helm/tacito-square-infra/` renders valid YAML.
   - Template output contains expected resources for all 7 sub-charts (nats, redis, postgresql, qdrant, otel-collector, keycloak, minio).
   - For each sub-chart, `helm template --set <component>.enabled=false` excludes that component from output.
   - `tools/helm/tacito-square/Chart.yaml` has no `dependencies` entries (or empty array).
2. Add `make test-helm-infra` target to Makefile that runs this script.
3. Run the script — it MUST FAIL (chart doesn't exist yet).

## Acceptance Criteria

1. `test/helm/test_infra_chart.sh` exists and is executable.
2. Running the script produces clear FAIL output for each check.
3. `make test-helm-infra` target exists and runs the script.
