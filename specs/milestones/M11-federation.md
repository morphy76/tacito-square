# Milestone M11: Federation

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

External A2A federation via HTTP gateway and external agent registries.

## Deliverable

External agent federation via A2A protocol and centralized external agent management registry.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M11.1 | A2A HTTP Gateway | keeper | SPEC-FR-M6.5 |
| SPEC-FR-M11.2 | External Agent Registry | keeper | SPEC-FR-M11.1 |
| SPEC-FR-M11.3 | Spawning MCP Servers using CRD from Keeper | keeper, operator | SPEC-FR-M3.10 |
| SPEC-FR-M11.4 | Thread Management | conversation-hub | SPEC-FR-M3.6, SPEC-FR-M6.0 |
