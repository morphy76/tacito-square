# Milestone M6: Policies & Governance

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Usage quotas enforced, default agents auto-spawn on community creation, HITL callback flows operational.

## Deliverable

Create community → default agents auto-spawn → HITL yield in conversation → human responds via callback → quotas enforced at API layer.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-14.1 | Default community agent (entry point) | FR-14.1 | keeper | SPEC-FR-05.1 |
| SPEC-FR-14.2 | Built-in agent archetypes | FR-14.2 | keeper | SPEC-FR-14.1 |
| SPEC-FR-14.3 | Community template with agent manifest | FR-14.3 | keeper | SPEC-FR-14.1, SPEC-FR-14.2 |
| SPEC-FR-15.1 | Community quotas | FR-15.1 | keeper | SPEC-FR-05.1 |
| SPEC-FR-15.2 | Agent quotas | FR-15.2 | keeper | SPEC-FR-01.1 |
| SPEC-FR-15.3 | Quota enforcement (Redis counters) | FR-15.3 | keeper | SPEC-FR-15.1, SPEC-NFR-CACHE |
| SPEC-FR-15.4 | Quota tracking & reporting | FR-15.4 | keeper | SPEC-FR-15.3 |
| SPEC-FR-04.7 | HITL yield | FR-04.7 | agent | SPEC-FR-04.1 |
| SPEC-FR-11.1 | HITL Agent Card flag | FR-11.1 | agent | SPEC-FR-06.1 |
| SPEC-FR-11.2 | HITL yield in reasoning | FR-11.2 | agent | SPEC-FR-04.7 |
| SPEC-FR-11.3 | HITL callback persistence | FR-11.3 | keeper | SPEC-FR-05.6 |
| SPEC-FR-11.4 | HITL human response | FR-11.4 | keeper | SPEC-FR-11.3 |
| SPEC-FR-11.5 | HITL TTL/escalation | FR-11.5 | keeper | SPEC-FR-11.3 |
| SPEC-FR-11.6 | HITL audit events | FR-11.6 | keeper | SPEC-FR-11.3 |
| SPEC-FR-05.4 | K8s NetworkPolicies | FR-05.4 | operator | SPEC-FR-05.1 |
