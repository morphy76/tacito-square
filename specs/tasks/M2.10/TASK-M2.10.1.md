# TASK-M2.10.1: Bitnami avoidance validation (RED)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.10.1                                |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.10                               |
| Phase         | RED                                         |
| Depends On    | none                                        |

## Description

Write a validation script that programmatically checks `tools/helm/tacito-square-infra/Chart.yaml` and `values.yaml` to verify zero references to Bitnami charts or repository URLs.

## Work Items

1. Create a validation script at `test/helm/test_no_bitnami.sh` that:
   - Scans `tools/helm/tacito-square-infra/Chart.yaml` for any occurrences of `charts.bitnami.com` or `bitnami` and fails if any are found.
   - Scans the rendered templates of the infrastructure Helm chart to ensure all generated resource containers utilize official free community or Red Hat base images (e.g. `postgres`, `redis:alpine`, `quay.io/keycloak`, `minio/minio`).
2. Run the script — verify it correctly fails on the current codebase.

## Acceptance Criteria

1. `test/helm/test_no_bitnami.sh` exists and is executable.
2. The script correctly identifies existing Bitnami dependencies in `tools/helm/tacito-square-infra/Chart.yaml` and fails.
