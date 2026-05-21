# Milestone M6: Communities & Messaging

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Agents form communities with hub-spoke topology, communicate via NATS messaging, support multi-threaded conversations, and can hand off conversations to other agents.

## Deliverable

Community with hub + 2 workers → user sends message → hub routes to worker → worker reasons and responds → conversation handoff between agents → specialist agent spawned on demand.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M6.1 | Community Topology (Hub-Spoke) | keeper, agent | SPEC-FR-M3.2 |
| SPEC-FR-M6.2 | NATS Inter-Agent Messaging | agent | SPEC-FR-M5.1 |
| SPEC-FR-M6.3 | NATS Subject Namespacing | agent, keeper | SPEC-FR-M6.2 |
| SPEC-FR-M6.4 | Thread Management | keeper | SPEC-FR-M3.2 |
| SPEC-FR-M6.5 | A2A Agent Cards | agent | SPEC-FR-M5.1 |
| SPEC-FR-M6.6 | Conversation Handoff | agent | SPEC-FR-M6.2, SPEC-FR-M5.3 |
| SPEC-FR-M6.7 | Specialist Agent Spawn | keeper, agent | SPEC-FR-M6.1, SPEC-FR-M3.6 |
