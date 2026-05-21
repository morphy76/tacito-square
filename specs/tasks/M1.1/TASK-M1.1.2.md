# TASK-M1.1.2: Create infrastructure Helm chart (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M1.1.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M1.1                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M1.1.1                                 |

## Description

Create the infrastructure Helm chart with all 7 sub-chart dependencies and dev-friendly defaults. This is the GREEN phase — implement the minimum to make the RED tests pass.

## Work Items

1. Create `tools/helm/tacito-square-infra/Chart.yaml`:
   - `apiVersion: v2`, `name: tacito-square-infra`, `type: application`
   - Dependencies: nats, redis, postgresql, qdrant, otel-collector, keycloak, minio
   - Each dependency with `condition: <component>.enabled`
2. Create `tools/helm/tacito-square-infra/values.yaml`:
   - All sub-charts enabled by default (`<component>.enabled: true`)
   - Dev defaults: no auth, standalone architecture, small persistence (1Gi)
   - Redis: `auth.enabled: false`, `architecture: standalone`
   - PostgreSQL: dev credentials (`keeper/keeper-dev`), database `tacito_keeper`
   - OTel Collector: OTLP receivers (gRPC 4317, HTTP 4318), debug exporter
   - MinIO: `enabled: false` (opt-in), dev credentials
3. Create `tools/helm/tacito-square-infra/templates/NOTES.txt` with post-install info.
4. Create `tools/helm/tacito-square-infra/templates/_helpers.tpl` with chart helpers.
5. Run `helm dependency update tools/helm/tacito-square-infra/`.
6. Verify `helm lint` and `helm template` pass.

## Acceptance Criteria

1. `helm lint tools/helm/tacito-square-infra/` passes.
2. `helm template tacito-infra tools/helm/tacito-square-infra/` renders manifests for all enabled services.
3. `helm dependency update` downloads all sub-chart archives.
