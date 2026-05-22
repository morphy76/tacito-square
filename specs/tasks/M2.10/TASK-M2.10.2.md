# TASK-M2.10.2: Migrate dependencies away from Bitnami (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.10.2                                |
| Status        | COMPLETE                                    |
| Spec          | SPEC-FR-M2.10                               |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.10.1                                |

## Description

Migrate all Bitnami dependencies (redis, postgresql, keycloak, minio) in the infrastructure Helm chart to official, free, non-commercial community charts.

## Work Items

1. Update `tools/helm/tacito-square-infra/Chart.yaml` to remove Bitnami chart dependencies.
2. Replace them with standard community alternatives:
   - **PostgreSQL**: e.g., using official postgresql from community-supported Helm repos or direct custom resource deployments.
   - **Redis**: e.g., using Redis community charts.
   - **Keycloak**: e.g., using official Keycloak charts or deployments.
   - **MinIO**: e.g., using the official MinIO Helm repository (`https://charts.min.io/`).
3. Update `values.yaml` configurations to conform to the new community chart formats.
4. Run `make helm-infra-deps` to refresh local dependencies.
5. Run `test/helm/test_no_bitnami.sh` — it MUST pass successfully.

## Acceptance Criteria

1. `Chart.yaml` contains zero references to Bitnami charts.
2. `make helm-infra-deps` successfully resolves all new dependencies.
3. `test/helm/test_no_bitnami.sh` passes successfully.
