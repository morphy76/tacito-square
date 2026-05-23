# SPEC-FR-M7.4: OIDC Login Flow (Keycloak)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.4                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M7.1, SPEC-FR-M3.9                 |
| Supersedes    | none                                        |

## Context

Both UIs authenticate users via OIDC authorization code flow through Keycloak.

## Specification

1. The BFF MUST implement OIDC authorization code flow with PKCE for both UIs.
2. The BFF MUST handle token refresh transparently.
3. The BFF MUST manage user sessions (cookie-based, HTTP-only, secure).
4. Role-based UI rendering MUST be driven by JWT claims.
5. The BFF MUST use the `tacito-ui` client configured in Keycloak.
6. Logout MUST invalidate both the local session and the Keycloak session.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
