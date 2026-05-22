# SPEC-FR-M2.1: Application Helm Chart (infra-free, binding interfaces)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M2.1                                |
| Status        | IMPLEMENTED                                       |
| Milestone     | M2                                          |
| Component     | deploy                                      |
| Depends On    | SPEC-FR-M1.1                                |
| Supersedes    | none                                        |

## Context

The application Helm chart deploys only the Tacito Square components (keeper, agent, operator, bff). It must not bundle infrastructure sub-charts — instead, it exposes binding interfaces (environment variables) for connecting to externally-managed infrastructure services.

## Specification

1. The chart at `tools/helm/tacito-square/` MUST NOT declare any infrastructure sub-chart dependencies.
2. The `values.yaml` MUST expose binding interfaces for each infrastructure service via component environment variables following the `TS_<COMPONENT>_<SERVICE>_*` naming convention.
3. The chart MUST include Helm templates for each component:
   - Keeper: Deployment, Service, ServiceAccount
   - Agent: Deployment (template, `replicaCount: 0` by default — agents spawned dynamically), Service
   - Operator: Deployment, ServiceAccount, ClusterRole, ClusterRoleBinding (conditional: `operator.enabled`)
   - BFF: Deployment, Service (conditional: `bff.enabled`)
4. Each Deployment MUST configure health probes (`/healthz` for liveness, `/readyz` for readiness).
5. The chart MUST include CRD templates for `TacitoAgent` and `TacitoCommunity` (installed via `crds/` directory or pre-install hook).
6. The chart MUST pass `helm lint` and `helm template` without errors.
7. Component image tags MUST default to the corresponding `VERSION.<component>` file values.

## Acceptance Criteria

1. `helm lint tools/helm/tacito-square/` passes without errors.
2. `helm template tacito tools/helm/tacito-square/` renders Deployment + Service for keeper, agent, operator, bff.
3. `Chart.yaml` has zero `dependencies` entries.
4. Setting `operator.enabled=false` or `bff.enabled=false` excludes those components.
5. All Deployments reference health probes at `/healthz` and `/readyz`.
6. CRD manifests are rendered for `TacitoAgent` and `TacitoCommunity`.

## Test Plan

1. **Lint**: `helm lint tools/helm/tacito-square/` — pass.
2. **Template**: `helm template tacito tools/helm/tacito-square/` — valid YAML, grep for all expected resources.
3. **No infra**: `grep -c 'dependencies' tools/helm/tacito-square/Chart.yaml` — expect 0 or empty array.
4. **Conditional**: `helm template --set operator.enabled=false` — operator Deployment absent.
5. **Deploy**: After M2.3–M2.6 (hello world builds), `helm install` on minikube with infra chart pre-installed — all component pods healthy.

## Files Affected

- `tools/helm/tacito-square/Chart.yaml` (MODIFY — remove dependencies)
- `tools/helm/tacito-square/values.yaml` (MODIFY — restructure for binding interfaces only)
- `tools/helm/tacito-square/templates/keeper-deployment.yaml` (NEW)
- `tools/helm/tacito-square/templates/keeper-service.yaml` (NEW)
- `tools/helm/tacito-square/templates/agent-deployment.yaml` (NEW)
- `tools/helm/tacito-square/templates/agent-service.yaml` (NEW)
- `tools/helm/tacito-square/templates/operator-deployment.yaml` (NEW)
- `tools/helm/tacito-square/templates/bff-deployment.yaml` (NEW)
- `tools/helm/tacito-square/templates/bff-service.yaml` (NEW)
- `tools/helm/tacito-square/crds/` (NEW — CRD manifests)
