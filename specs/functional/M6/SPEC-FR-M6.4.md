# SPEC-FR-M6.4: Thread Management

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M6.4                                |
| Status        | DRAFT                                       |
| Milestone     | M6                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.2                                |
| Supersedes    | none                                        |

## Context

Conversations within communities are organized into threads for multi-turn engagement tracking. A community can have multiple concurrent threads.

## Specification

1. The system MUST define a `Thread` entity with fields: ID, community reference, status (active, paused, completed), participants (agent IDs), created/updated timestamps.
2. The system MUST expose CRUD REST endpoints at `GET/POST /api/v1/communities/{community_id}/threads` and `GET/PUT/DELETE /api/v1/communities/{community_id}/threads/{id}`.
3. A community MUST support multiple concurrent active threads.
4. Thread creation MUST specify the initial message and target agent (defaults to hub).
5. Thread state MUST be persisted in PostgreSQL.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
