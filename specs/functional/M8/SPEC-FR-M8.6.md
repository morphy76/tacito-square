# SPEC-FR-M8.6: Prompt Management (CRUD + versioning)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.6                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.8, SPEC-FR-M3.4                 |
| Supersedes    | none                                        |

## Context

Prompts define agent personas and instructions. Building upon the foundational prompt template and collection models defined in SPEC-FR-M3.4, this specification introduces advanced versioning, Redis caching, and route protection to support secure enterprise governance.

## Specification

1. The system MUST define a `Prompt` aggregate: name, content (template), version, status (draft, active, archived).
2. CRUD REST endpoints at `GET/POST /api/v1/prompts` and `GET/PUT/DELETE /api/v1/prompts/{id}`.
3. Updates MUST create a new version (immutable versions, per constitution P6).
4. Support prompt import/export in JSON format.
5. Prompt resolution at spawn MUST select the latest active version.
6. Prompts MUST be cached in Redis (per SPEC-NFR-CACHE).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
