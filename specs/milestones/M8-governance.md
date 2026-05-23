# Milestone M8: Governance

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

RBAC enforcement, usage quotas, human-in-the-loop (HITL) flows, audit trail, and management of prompts and skills.

## Deliverable

Role-based access → quota-enforced agent spawning → HITL yield/callback → audit trail queries → managed prompts and skills resolved at agent spawn.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M8.1 | RBAC Role Model & Route Protection | keeper, shared | SPEC-FR-M3.9 |
| SPEC-FR-M8.2 | Usage Quotas (community + agent) | keeper | SPEC-FR-M3.6 |
| SPEC-FR-M8.3 | Quota Enforcement (Redis counters) | keeper | SPEC-FR-M8.2, SPEC-NFR-CACHE |
| SPEC-FR-M8.4 | HITL Yield & Callback Flows | agent, keeper | SPEC-FR-M5.2, SPEC-FR-M6.4 |
| SPEC-FR-M8.5 | Audit Trail (events + queries) | keeper | SPEC-FR-M3.8 |
| SPEC-FR-M8.6 | Prompt Management (CRUD + versioning) | keeper | SPEC-FR-M3.8, SPEC-FR-M3.4 |
| SPEC-FR-M8.7 | Skills Management (CRUD + MCP attach) | keeper | SPEC-FR-M3.8, SPEC-FR-M3.3 |
