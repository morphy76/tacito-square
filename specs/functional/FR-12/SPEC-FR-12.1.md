# SPEC-FR-12.1: Bearer JWT Authentication

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-12.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-12.1                            |
| Component     | shared                             |
| Depends On    | SPEC-FR-08.5                       |

## Context

All Tacito Square APIs are protected by Bearer JWT authentication. Users and services are not constrained to a UI — they interact directly via authenticated REST APIs.

## Specification

1. Every HTTP API endpoint (except `/healthz`, `/readyz`) MUST require a valid `Authorization: Bearer <token>` header.
2. Token extraction MUST be handled by the shared `auth` package.
3. Invalid or missing tokens MUST result in HTTP 401 Unauthorized.
4. Token validation against Keycloak JWKS is deferred to middleware (M6).
5. Claims extracted from the token MUST be propagated via Go context.

## Acceptance Criteria

1. Bearer token extraction works correctly (see SPEC-FR-08.5)
2. Claims are available in context for downstream use
3. Health endpoints excluded from auth requirement

## Files

- `internal/shared/auth/auth.go` ✅ IMPLEMENTED
