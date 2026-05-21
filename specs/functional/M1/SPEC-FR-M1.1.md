# SPEC-FR-M1.1: Infrastructure Helm Chart

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M1.1                                |
| Status        | DRAFT                                       |
| Milestone     | M1                                          |
| Component     | deploy                                      |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

Tacito Square relies on several infrastructure services (NATS, Redis, PostgreSQL, Qdrant, OTel Collector, Keycloak, MinIO). These must be deployed separately from the application components to enable clean separation of concerns: the application chart binds to infrastructure via connection parameters, and infrastructure can be swapped for managed services in production.

## Specification

1. The system MUST provide a dedicated Helm chart at `tools/helm/tacito-square-infra/`.
2. The chart MUST declare sub-chart dependencies for:
   - **NATS** (nats-io Helm chart, messaging backbone)
   - **Redis** (Bitnami Helm chart, STM + cache)
   - **PostgreSQL** (Bitnami Helm chart, keeper persistence)
   - **Qdrant** (official Helm chart, vector storage)
   - **OpenTelemetry Collector** (official Helm chart, tracing)
   - **Keycloak** (Bitnami Helm chart, OIDC provider)
   - **MinIO** (Bitnami Helm chart, S3-compatible storage)
3. Each sub-chart dependency MUST be conditionally enabled via `<component>.enabled` (default: `true` for dev).
4. The `values.yaml` MUST provide sensible dev defaults (no auth, small persistence volumes, single replicas).
5. The chart MUST NOT contain any application component templates (keeper, agent, operator, bff).
6. The chart MUST pass `helm lint` and `helm template` without errors.
7. The Keycloak configuration MUST include the `tacito` realm with pre-configured clients (`tacito-keeper`, `tacito-ui`) and roles (`keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`).

## Acceptance Criteria

1. `helm lint tools/helm/tacito-square-infra/` passes without errors.
2. `helm template tacito-infra tools/helm/tacito-square-infra/` renders valid manifests for all 7 infrastructure services.
3. `helm install tacito-infra tools/helm/tacito-square-infra/` on a minikube cluster starts all services and they become healthy within 5 minutes.
4. Disabling any single sub-chart via `--set <component>.enabled=false` excludes that service from rendering.
5. The application Helm chart (`tools/helm/tacito-square/`) has zero sub-chart dependencies on infrastructure after this change.

## Test Plan

1. **Lint**: `helm lint tools/helm/tacito-square-infra/` — must pass.
2. **Template**: `helm template tacito-infra tools/helm/tacito-square-infra/` — must render valid YAML.
3. **Conditional**: For each infra component, run `helm template --set <component>.enabled=false` and verify the component is absent from output.
4. **Deploy**: `helm install tacito-infra tools/helm/tacito-square-infra/ --wait --timeout 5m` on minikube — all pods healthy.
5. **Isolation**: Verify `tools/helm/tacito-square/Chart.yaml` has no `dependencies` entries after refactoring.

## Files Affected

- `tools/helm/tacito-square-infra/Chart.yaml` (NEW)
- `tools/helm/tacito-square-infra/values.yaml` (NEW)
- `tools/helm/tacito-square-infra/templates/` (NEW — NOTES.txt, _helpers.tpl)
- `tools/helm/tacito-square/Chart.yaml` (MODIFY — remove all infrastructure dependencies)
- `tools/helm/tacito-square/values.yaml` (MODIFY — remove infrastructure configuration sections)
