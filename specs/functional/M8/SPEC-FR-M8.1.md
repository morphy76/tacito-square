# SPEC-FR-M8.1: OIDC/JWT Authentication

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.1                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper, shared                              |
| Depends On    | SPEC-FR-M2.2                                |
| Supersedes    | none                                        |

## Context

All keeper API endpoints must be protected by OIDC/JWT tokens issued by Keycloak. The authentication middleware lives in the shared library and is reusable across components.

## Specification

1. The system MUST implement a Gin middleware for JWT validation using the Zitadel OIDC library (per SPEC-NFR-STACK).
2. The middleware MUST discover JWKS endpoints automatically from the OIDC issuer URL.
3. The middleware MUST extract roles from JWT claims and make them available to downstream handlers.
4. The Keycloak `tacito` realm MUST be pre-configured with roles: `keeper-admin`, `keeper-viewer`, `user`, `agent-spawner`.
5. JWT claims (subject, roles) MUST be logged with every request (per SPEC-NFR-LOG).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
