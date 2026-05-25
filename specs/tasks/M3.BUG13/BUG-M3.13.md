# BUG-M3.13: Inconsistent REST API Behaviors, Prompts, and Skills

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.13                                                          |
| Status        | IMPLEMENTED                                                        |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | `internal/keeper/domain/model/*`, `internal/keeper/adapters/*`      |
| Violates      | SPEC-FR-M3.3, SPEC-FR-M3.4, SPEC-NFR-HTTP, SPEC-NFR-OPENAPI       |
| Discovered    | Manual API design audit and tenant feedback                        |

## Problem Statement

A design audit of the Milestone 3 HTTP REST endpoints, Prompt models, and Skill structures has identified several inconsistencies and architectural misalignments:

1. **Skill Model Inconsistency**: The current `Skill` aggregate references specific MCP servers and tools whitelists/blacklists directly. This conflicts with the agent-level custom configurations (`mcp_clients` JSON) and doesn't align with dynamic capability resolution.
2. **Missing Skill Collections**: Unlike prompts, which support cataloging via `PromptCollection`, skills are flat and lack grouping support via a comparable `SkillCollection` aggregate structure.
3. **Redundant Prompt Attributes**: The `PromptTemplate` model includes `role` and `version` fields, which are not desired or needed for the simplified prompt structures.
4. **Incorrect Return Values on PUT**: Standard REST endpoints currently return the *newly created/updated* state, whereas the specified workflow requires returning the *previous* state of the resource on successful  `PUT` (should be the original state prior to update).
5. **Inconsistent REST Status Codes**: Missing resources on actions like `DELETE` return `204 No Content` or `500 Internal Server Error` instead of an explicit `404 Not Found` response code. `POST` should return 201 on success and the location header to get the newly created resource

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| Prompt Model | `internal/keeper/domain/model/prompt.go` | Contains redundant `role` and `version` fields. |
| Skill Model | `internal/keeper/domain/model/skill.go` | Skill aggregate references MCPServers and tools instead of being a simple capability. Lacks a grouping `SkillCollection` model. |
| HTTP Handlers | `internal/keeper/adapters/inbound/http/*` | POST/PUT success returns new resource; Delete lacks proper 404 handling. |
| pgx Repositories | `internal/keeper/adapters/outbound/postgres/*` | Deletes do not check `RowsAffected` to bubble up "not found" errors. |
| Database Migrations | `deploy/postgres/migrations/*` | Table definitions for prompt templates contain role/version, skills contain tools, and skill collections table is missing. |

## Impact

1. **API Redundancy and Complexity**: Unneeded prompt metadata complicates client integrations and payload processing.
2. **Reduced Flexibility**: Inability to group skill capabilities limits dynamic profile assignments for agent deployments.
3. **REST Semantics Violations**: Non-standard HTTP status codes and return values violate strict system contracts and OIDC middleware assumptions.

## Expected Behaviour

1. **Skill Simplification**: `Skill` aggregate should represent a simple capability (no MCPServers or allowed/denied tools references).
2. **Skill Collections**: Add a `SkillCollection` aggregate in the same fashion as `PromptCollection`.
3. **Simplified Prompts**: Remove `role` and `version` from `PromptTemplate`.
4. **Response Payload Contracts**: Successful `POST` returns `nil` (JSON `null`); successful `PUT` returns the unmodified previous state of the resource.
5. **Status Codes**: Missing resource requests on `GET`, `PUT`, and `DELETE` return HTTP `404 Not Found` with standard `{ "error": "not found" }` message instead of silent success.

## Acceptance Criteria

1. Database schema migration runs successfully, updating `prompt_templates`, `skills`, and adding `skill_collections` / `skill_collection_skills`.
2. Unit tests verify that `PUT` requests return the resource's previous state.
3. Unit tests verify `404 Not Found` is correctly returned when deleting or updating a non-existent resource.
4. Unit tests verify `POST` requests return the location header and 201 status code.
5. Skill Collection CRUD endpoints fully work and can resolve active skills.
6. OpenAPI is properly synchd and verified by contract tests
