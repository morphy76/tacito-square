# SPEC-FR-M2.10: Avoid Bitnami (Leverage Free & Non-Commercial Infrastructural Dependencies)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.10                               |
| Status        | ACCEPTED                                    |
| Milestone     | M2                                          |
| Component     | deploy                                      |
| Depends On    | SPEC-FR-M1.1                                |
| Supersedes    | none                                        |

## Context

Bitnami charts and images (governed by VMware/Broadcom) include commercial tracking, licensing, and registry constraints that can restrict unrestricted non-commercial enterprise usage or lead to licensing overhead. To leverage 100% free, non-commercial, open-source community-governed or Red Hat-governed infrastructural dependencies, Tacito Square's infrastructure Helm chart (`tacito-square-infra`) must be migrated away from Bitnami charts (specifically `bitnami/redis`, `bitnami/postgresql`, `bitnami/keycloak`, and `bitnami/minio`) in favor of official community chart repositories or custom manifests leveraging official open images (e.g. Docker Hub or Red Hat Quay).

## Specification

1. **Eliminate Bitnami Charts**: The infrastructure chart `tools/helm/tacito-square-infra/Chart.yaml` MUST NOT contain any dependency repository entries pointing to `https://charts.bitnami.com/bitnami`.
2. **Free Community Chart/Image Alternatives**:
   - **PostgreSQL**: Migrate from `bitnami/postgresql` to a standard free community chart (such as the CloudNativePG operator, or the official standard community Helm chart) using the official `postgres` Docker Hub/RedHat registry images.
   - **Redis**: Migrate from `bitnami/redis` to the official Redis operator/chart (e.g. `https://ot-container-kit.github.io/helm-charts` or standard community deployments) using official `redis:alpine` Docker Hub images.
   - **Keycloak**: Migrate from `bitnami/keycloak` to the official Keycloak community Helm chart or standard Keycloak Operator using Keycloak's official Red Hat UBI (Universal Base Image) based containers (`quay.io/keycloak/keycloak`).
   - **MinIO**: Migrate from `bitnami/minio` to the official MinIO community Helm chart (`https://charts.min.io/`) running official `minio/minio` Docker Hub containers.
3. **No Commercial Overhead**: All substituted charts/images MUST be validated as fully free, non-commercial open-source community releases without Broadcom/VMware enterprise registration requirements.
4. **Liveness & Readiness Alignment**: Ensure all new community chart installations preserve the readiness/liveness ports and binding DNS names so the application components (defined in `SPEC-FR-M2.1`) can connect seamlessly.

## Acceptance Criteria

1. `tools/helm/tacito-square-infra/Chart.yaml` contains zero references to `https://charts.bitnami.com/bitnami`.
2. All replacement charts are pulled from official, non-commercial community repositories.
3. The entire infrastructure stack (`tacito-square-infra`) can be linted and templated without error via Helm.
4. Standard binding hostnames (e.g., `tacito-infra-postgresql`, `tacito-infra-redis`) remain functional or are cleanly re-bound in the application chart.

## Test Plan

1. **Lint**: Run `make helm-infra-lint` to ensure the updated infrastructure chart passes all validations.
2. **Template**: Run `make helm-infra-template` to ensure all resource manifests render correctly.
3. **No Bitnami References**: Run a recursive search on `tools/helm/tacito-square-infra/` to verify zero instances of `bitnami` registry or repo URLs.

## Files Affected

- `tools/helm/tacito-square-infra/Chart.yaml` (MODIFY)
- `tools/helm/tacito-square-infra/values.yaml` (MODIFY)
