# SPEC-FR-14: Default & Built-in Agents

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-14                         |
| Status        | DRAFT                              |
| Milestone     | M4                                 |
| Component     | keeper, agent                      |
| Depends On    | SPEC-FR-05.1                       |

## Context

Communities need predictable structure at creation time. A **default agent** serves as the community entry point (the hub in Hub-Spoke). **Built-in agents** are archetype agents (planner, summarizer, router) that are auto-spawned when a community is created, providing immediate capabilities without manual configuration.

## Specification

### FR-14.1: Default Community Agent (Entry Point)
1. Every community MUST have exactly one **default agent** designated as the entry point.
2. The default agent MUST be auto-spawned when the community is created.
3. All inbound messages to the community MUST be routed through the default agent first.
4. The default agent's prompt and skills are defined in the community template.
5. If the default agent terminates, the community MUST enter `degraded` state.

### FR-14.2: Built-in Agent Archetypes
1. The system MUST define a registry of built-in agent archetypes:
   - `planner` — decomposes complex tasks into sub-tasks and delegates
   - `summarizer` — condenses thread history and produces summaries
   - `router` — analyzes intent and routes to the best specialist
   - `guardian` — monitors conversation for policy compliance
2. Each archetype MUST have a default prompt template and skill set.
3. Archetypes are shipped as seed data (prompt + skills) via DB migrations.
4. Custom archetypes MAY be added via the Prompt & Skills API.

### FR-14.3: Community Template with Agent Manifest
1. A community template MUST define:
   - `default_agent`: archetype or custom prompt/skills for the entry point
   - `built_in_agents`: list of archetypes to auto-spawn at community creation
   - `topology`: Hub-Spoke (default), Mesh, Pipeline
   - `quotas`: resource limits (see FR-15)
2. When a community is created from a template, the Keeper MUST:
   a. Create the community entity
   b. Spawn the default agent as hub
   c. Spawn each built-in agent from the manifest
   d. Wire NATS subject bindings
3. Templates are persisted in PostgreSQL and exposed via REST API.

## Acceptance Criteria

1. Creating a community auto-spawns default agent as entry point
2. Built-in agents (e.g., planner) are auto-spawned from manifest
3. Messages to community are routed through default agent
4. Default agent termination triggers community degraded state
5. Archetype seed data is available after migration
6. Community template CRUD via REST API

## Test Plan

- Unit: Community template validation, manifest parsing
- Unit: Auto-spawn orchestration with mock ports
- Integration: Community creation → verify agents spawned (testcontainers)

## Files Affected

- `internal/keeper/domain/community_template.go` (NEW)
- `internal/keeper/domain/archetype.go` (NEW)
- `internal/keeper/service/community_service.go` (MODIFY)
- `migrations/` — archetype seed data (NEW)
