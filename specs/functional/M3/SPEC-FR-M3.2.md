# SPEC-FR-M3.2: Community Domain Model & CRUD API

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.2                                |
| Status        | DRAFT                                       |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M2.3                                |
| Supersedes    | none                                        |

## Context

Communities are the organizational unit for agents. A community groups one or more agents under a topology (initially hub-spoke) and provides isolation boundaries. Keeper needs a domain model and CRUD API for community management.

## Specification

1. The system MUST define a `Community` aggregate in the keeper domain layer with fields: name (unique), description, topology (enum: hub-spoke), configuration, status (created, active, suspended, terminated).
2. The system MUST expose CRUD REST endpoints at `GET/POST /api/v1/communities` and `GET/PUT/DELETE /api/v1/communities/{id}`.
3. The domain layer MUST NOT import adapter or application packages (per SPEC-NFR-HEXAGONAL).
4. Communities MUST be the unit of isolation — agents in different communities SHOULD NOT communicate unless explicitly federated.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
