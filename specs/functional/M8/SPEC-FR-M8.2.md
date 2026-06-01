# SPEC-FR-M8.2: RBAC Role Model & Route Protection

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.2                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper, shared                              |
| Depends On    | SPEC-FR-M3.9, SPEC-FR-M8.1                  |
| Supersedes    | none                                        |

## Context

Keeper API endpoints must enforce role-based access control using Keycloak roles from JWT tokens.

## Specification

1. The system MUST define role-to-permission mappings: `keeper-admin` (full CRUD), `keeper-viewer` (read-only), `agent-spawner` (create agents + assign), `user` (interact via threads).
2. The system MUST implement a Gin authorization middleware checking required roles per route.
3. Route protection MUST be declarative (configuration-based, not hardcoded).
4. Unauthorized requests MUST return 403 with standard error response.
5. Principal identity (subject + roles) MUST be logged with every request.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
