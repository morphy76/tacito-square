# SPEC-FR-M3.1: Agent Domain Model & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.1                                |
| Status        | DRAFT                                       |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3                                |
| Supersedes    | none                                        |

## Context

Keeper needs a domain model for agents and REST API endpoints to create, read, update, and delete agent definitions. Agents exist as definitions managed by keeper; runnable instances are created only when assigned to a community.

## Specification

1. The system MUST define an `Agent` aggregate in the keeper domain layer with fields: name (unique), description, LLM configuration (model, temperature, max tokens), system prompt reference, status (defined, assigned, active, terminated).
2. The system MUST expose CRUD REST endpoints at `GET/POST /api/v1/agents` and `GET/PUT/DELETE /api/v1/agents/{id}`.
3. The domain layer MUST NOT import adapter or application packages (per SPEC-NFR-HEXAGONAL).
4. Input validation MUST use Gin binding tags (per SPEC-NFR-HTTP).
5. All responses MUST use the standard JSON error format on failure.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
