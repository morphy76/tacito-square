# Milestone M7: K8s Operator

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Agent lifecycle managed via CRDs instead of direct K8s API calls.

## Deliverable

`kubectl apply -f agent.yaml` → Operator creates full agent deployment with security policies and HPA.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-07.1 | Agent CRD | FR-07.1 | operator | — |
| SPEC-FR-07.2 | AgentCommunity CRD | FR-07.2 | operator | SPEC-FR-07.1 |
| SPEC-FR-07.3 | Validating webhooks | FR-07.3 | operator | SPEC-FR-07.1 |
| SPEC-FR-07.4 | Mutating webhooks | FR-07.4 | operator | SPEC-FR-07.1 |
| SPEC-FR-M7-INT | Keeper→Operator integration | — | keeper, operator | SPEC-FR-07.1 |
| SPEC-NFR-HPA | Horizontal Pod Autoscaling | NFR-HPA | operator, deploy | SPEC-FR-07.1 |
| SPEC-FR-15-M7 | Quota enforcement at operator level | FR-15.3 | operator | SPEC-FR-15.3 |
