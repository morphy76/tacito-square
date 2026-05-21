# TASK-M1.1.4: Decouple infrastructure from application chart (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.1.4                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.1                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M1.1.3                                 |

## Description

Remove all infrastructure sub-chart dependencies from the application Helm chart. The application chart must only contain binding interfaces (environment variables) for connecting to externally-managed infrastructure.

## Work Items

1. Remove all `dependencies` entries from `tools/helm/tacito-square/Chart.yaml` (nats, redis, postgresql, qdrant, otel-collector, keycloak, minio).
2. Remove infrastructure configuration sections from `tools/helm/tacito-square/values.yaml`:
   - Remove `nats:`, `redis:`, `postgresql:`, `qdrant:`, `otel-collector:`, `keycloak:`, `minio:` top-level keys.
3. Keep binding interface environment variables in component sections (e.g., `TS_KEEPER_DB_HOST`, `TS_KEEPER_NATS_URL`, etc.) — these are how the app chart connects to infrastructure.
4. Remove `charts/` directory lock files if present.
5. Verify `helm lint tools/helm/tacito-square/` passes with no dependencies.

## Acceptance Criteria

1. `tools/helm/tacito-square/Chart.yaml` has no `dependencies` entries.
2. `tools/helm/tacito-square/values.yaml` has no infrastructure sub-chart configuration.
3. Component sections still contain `TS_*` environment variables for infrastructure binding.
4. `helm lint tools/helm/tacito-square/` passes.
