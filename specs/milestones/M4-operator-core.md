# Milestone M4: Operator Core

| Field      | Value |
|------------|-------|
| Status     | ✔️ IMPLEMENTED |

## Goal

The Operator watches `TacitoAgent` CRDs and instantiates or destroys agent Deployments and headless Services accordingly. Supports operator-managed zero-scaling of idle agents.

Communities are **not** deployable Kubernetes units and have no CRD. Community lifecycle operations (`/deploy`, `/undeploy`) are Keeper REST API calls that bulk-orchestrate `TacitoAgent` CRDs, not Operator-reconciled resources.

## Deliverable

`kubectl apply -f agent.yaml` → Operator creates agent Deployment + headless Service → agent pod starts and becomes healthy. `kubectl delete` → pod and service removed. Idle agents scale to zero replicas.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M4.1 | Agent CRD Definition & Registration | operator | none |
| ~~SPEC-FR-M4.2~~ | ~~AgentCommunity CRD Definition~~ | ~~operator~~ | _REJECTED — communities are Keeper DB entities, not K8s CRDs_ |
| SPEC-FR-M4.3 | Reconciliation Controller | operator | SPEC-FR-M4.1 |
| SPEC-FR-M4.4 | Zero-Scaling Support | operator | SPEC-FR-M4.3 |
| SPEC-FR-M4.6 | Agent CRD Submission | keeper | SPEC-FR-M3.7, SPEC-FR-M4.1 |
| SPEC-FR-M4.7 | Agent & Community Lifecycle Management REST API | keeper, operator | SPEC-FR-M3.7, SPEC-FR-M4.1, SPEC-FR-M4.6 |
