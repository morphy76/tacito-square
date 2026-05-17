# SPEC-FR-08.5: OIDC/JWT Authentication

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-08.5                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-08.5                            |
| Component     | shared                             |
| Depends On    | —                                  |

## Context

All API endpoints require Bearer JWT authentication. Tokens are issued by Keycloak and validated by the API layer.

## Specification

1. `Claims` struct MUST contain `Subject`, `Email`, and `Roles` fields.
2. `ExtractBearerToken(authHeader)` MUST:
   a. Return error if header is empty
   b. Return error if format is not `Bearer <token>`
   c. Return error if token is empty after "Bearer "
   d. Return the trimmed token on success
3. `ContextWithClaims(ctx, claims)` stores claims in context.
4. `ClaimsFromContext(ctx)` retrieves claims, returns nil if absent.
5. All API routes MUST require a valid Bearer token (enforced by middleware in M6).

## Acceptance Criteria

1. Empty header → error "missing authorization header"
2. "Basic xxx" → error "invalid authorization header format"
3. "Bearer " (empty token) → error "empty bearer token"
4. "Bearer valid-token-123" → returns "valid-token-123"
5. Claims round-trip through context
6. ClaimsFromContext returns nil for empty context
7. Case-insensitive "bearer" accepted

## Files

- `internal/shared/auth/auth.go` ✅ IMPLEMENTED
- `internal/shared/auth/auth_test.go` ✅ 7 tests passing
