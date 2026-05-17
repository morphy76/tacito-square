# Milestone M4: Prompt & Skills Management

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Keeper manages prompts and skills; agents are spawned with resolved configurations.

## Deliverable

Create prompt + skills via API → spawn agent with resolved config → agent uses correct persona and tools.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-02.1 | Prompt CRUD domain & service | FR-02.1 | keeper | — |
| SPEC-FR-02.2 | Prompt versioning | FR-02.2 | keeper | SPEC-FR-02.1 |
| SPEC-FR-02.3 | Prompt import/export | FR-02.3 | keeper | SPEC-FR-02.1 |
| SPEC-FR-03.1 | Skill CRUD domain & service | FR-03.1 | keeper | — |
| SPEC-FR-03.2 | MCP tool attach/detach | FR-03.2 | keeper | SPEC-FR-03.1 |
| SPEC-FR-03.3 | Skill assignment at spawn | FR-03.3 | keeper | SPEC-FR-03.1, SPEC-FR-01.1 |
| SPEC-FR-M4-PG | PostgreSQL: prompts & skills persistence | FR-10.2 | keeper | SPEC-FR-02.1, SPEC-FR-03.1 |
| SPEC-FR-M4-API | Keeper HTTP API: prompts & skills | FR-08.1 | keeper | SPEC-FR-02.1, SPEC-FR-03.1 |
| SPEC-FR-M4-SPAWN | Spawn flow integration (resolve prompt + skills → config) | FR-01.1 | keeper | ALL above |
