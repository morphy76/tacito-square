# Milestone M5: Communities & Messaging

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Multi-agent communities with threads, NATS messaging, Hub-Spoke topology, and audit trail.

## Deliverable

Community with concurrent threads, 3 agents (1 hub, 2 workers), conversation handoff between agents, audit log.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-05.1 | Community domain & service | FR-05.1 | keeper | — |
| SPEC-FR-05.2 | Hub-Spoke topology | FR-05.2 | agent, keeper | SPEC-FR-05.1 |
| SPEC-FR-05.3 | NATS subject namespacing | FR-05.3 | agent, keeper | SPEC-FR-05.1 |
| SPEC-FR-05.5 | Multi-thread engagements | FR-05.5 | keeper | SPEC-FR-05.1 |
| SPEC-FR-05.6 | Thread CRUD | FR-05.6 | keeper | SPEC-FR-05.5 |
| SPEC-FR-06.1 | A2A Agent Cards | FR-06.1 | agent | SPEC-FR-05.1 |
| SPEC-FR-06.2 | NATS internal messaging | FR-06.2 | agent | SPEC-FR-05.3 |
| SPEC-FR-06.4 | Hub routing | FR-06.4 | agent | SPEC-FR-05.2 |
| SPEC-FR-04.5 | Specialist agent spawn | FR-04.5 | keeper, agent | SPEC-FR-01.1 |
| SPEC-FR-04.6 | Conversation handoff | FR-04.6 | agent | SPEC-FR-04.2 |
| SPEC-FR-01.6 | Audit log per transition | FR-01.6 | keeper | — |
| SPEC-FR-09.4 | Audit log queries | FR-09.4 | keeper | SPEC-FR-01.6 |
