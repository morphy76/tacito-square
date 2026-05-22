# TASK-M2.1.2: Create component Helm templates (GREEN)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | TASK-M2.1.2                                 |
| Status        | TODO                                        |
| Spec          | SPEC-FR-M2.1                                |
| Phase         | GREEN                                       |
| Depends On    | TASK-M2.1.1                                 |

## Description

Create Helm templates for all 4 application components. Each component needs a Deployment and Service (where applicable), with health probes, environment variable injection from values, and conditional rendering.

## Work Items

1. Create/update `templates/keeper-deployment.yaml`:
   - Deployment with image from `keeper.image`, env from `keeper.env`, resources from `keeper.resources`.
   - Liveness probe: `httpGet /healthz`, readiness probe: `httpGet /readyz`.
   - ServiceAccount reference.
2. Create `templates/keeper-service.yaml`: ClusterIP Service on `keeper.service.port`.
3. Create `templates/keeper-serviceaccount.yaml`.
4. Create `templates/agent-deployment.yaml`:
   - Default `replicaCount: 0` (agents spawned dynamically).
   - Health probes on `/healthz` and `/readyz`.
5. Create `templates/agent-service.yaml`: headless Service.
6. Create `templates/operator-deployment.yaml` (conditional: `operator.enabled`).
7. Create `templates/bff-deployment.yaml` (conditional: `bff.enabled`).
8. Create `templates/bff-service.yaml` (conditional: `bff.enabled`).
9. Update `templates/_helpers.tpl` with common labels and selectors.

## Acceptance Criteria

1. `helm template` renders Deployment + Service for keeper.
2. `helm template` renders Deployment for agent with `replicas: 0`.
3. `helm template --set operator.enabled=true` renders operator Deployment.
4. `helm template --set bff.enabled=true` renders BFF Deployment + Service.
5. All Deployments configure `/healthz` and `/readyz` probes.
