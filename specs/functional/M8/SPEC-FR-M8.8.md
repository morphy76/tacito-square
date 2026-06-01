# SPEC-FR-M8.8: Skills Management (CRUD + MCP attach)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.8                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.8, SPEC-FR-M3.3                |
| Supersedes    | none                                        |

## Context

Skills are bundles of MCP tools that can be attached to agents. Building upon the foundational skill model defined in SPEC-FR-M3.3, this specification implements advanced authorization controls and performance-optimized caching for runtime tool resolution.

## Specification

1. The system MUST define a `Skill` aggregate: name, description, MCP server URL, tool list, status.
2. CRUD REST endpoints at `GET/POST /api/v1/skills` and `GET/PUT/DELETE /api/v1/skills/{id}`.
3. Attach/detach skills: `POST/DELETE /api/v1/agents/{agent_id}/skills/{skill_id}`.
4. At spawn, attached skills MUST be resolved and injected as MCP server endpoints.
5. Skill descriptors MUST be cached in Redis (per SPEC-NFR-CACHE).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
