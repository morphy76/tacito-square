# Milestone M4: Operator Core

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Operator watches TacitoAgent and TacitoCommunity CRDs and instantiates/destroys agent pods accordingly. Supports operator-managed zero-scaling.

## Deliverable

`kubectl apply -f agent.yaml` → operator creates agent Deployment + Service → agent pod starts and becomes healthy. `kubectl delete` → pod removed. Idle agents scale to zero replicas.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M4.1 | Agent CRD Definition & Registration | operator | none |
| SPEC-FR-M4.2 | AgentCommunity CRD Definition | operator | SPEC-FR-M4.1 |
| SPEC-FR-M4.3 | Reconciliation Controller | operator | SPEC-FR-M4.1 |
| SPEC-FR-M4.4 | Zero-Scaling Support | operator | SPEC-FR-M4.3 |
