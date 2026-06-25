# TASK-M7.2-T7: BFF Dual-Authentication Middleware (Session Cookie & Bearer Token)

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T7                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | BFF (`internal/bff/`)                                              |
| Status      | VERIFIED                                                           |
| Depends On  | SPEC-FR-M7.1, SPEC-FR-M7.4                                         |

## Objective

Enhance the BFF authentication middleware to support dual authentication modes:
1. **OIDC Session Cookie**: Traditional browser cookie-based authentication with transparent token renewal using Redis and Keycloak refresh tokens.
2. **Stateless HTTP Bearer Token**: Standard `Authorization: Bearer <JWT>` header validation using Keycloak JWKS keys. When a Bearer token is present, validate it statelessly. In this mode, no session tracking or token renewal is performed by the BFF.

This allows CLI tools, developers, and alternative API clients to interact with the configurator APIs directly using tokens while the web browser continues to use secure cookies.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/middleware.go` | MODIFY |
| `internal/bff/adapters/inbound/http/middleware_test.go` | MODIFY |

## RED Phase

1. **Dual Auth Middleware Test Suite**:
   - Write unit tests in `middleware_test.go` to assert:
     - **Valid Bearer Token**: A request with `Authorization: Bearer <valid_jwt>` succeeds, resolves roles/tenant ID, and does not check Redis or attempt token renewal.
     - **Expired Bearer Token**: A request with `Authorization: Bearer <expired_jwt>` is rejected with `401 Unauthorized` immediately (no renewal).
     - **Valid Session Cookie**: A request with a valid session cookie and no Bearer token succeeds, checking Redis.
     - **Expired Session Cookie with Valid Refresh**: A request with an expired session cookie triggers a successful Keycloak token refresh, updates Redis, and succeeds.
     - **Missing Both**: A request with neither is rejected with `401 Unauthorized`.
   - Run Go test suite: `make test` — must fail because the dual-auth logic is not yet implemented.

## GREEN Phase

1. **Implement JWT Validation Hook**:
   - Integrate Keycloak JWKS signature verification in the middleware using the OIDC client.
   - Cache JWKS keys in memory with a configurable TTL to prevent outbound calls on every request.

2. **Refactor Authentication Middleware**:
   - Check for the `Authorization` header. If it starts with `Bearer `:
     - Extract and validate the JWT.
     - If validation fails, return `401 Unauthorized` with standard JSON payload `{ "error": "Invalid bearer token" }`.
     - If validation succeeds, extract tenant ID and roles, populate the request context, and proceed. Skip Redis checks and token renewal.
   - If the header is missing:
     - Fall back to checking the session cookie.
     - Retrieve session from Redis. If access token is expired, trigger Keycloak refresh token flow.
     - If session is valid or successfully refreshed, populate request context and proceed.
     - If session is invalid, return `401 Unauthorized` or redirect to login.

3. **Verify tests**:
   - Run `make test` and confirm all middleware tests pass.

## REFACTOR Phase

- Ensure trace/span context is correctly populated with tenant details under both authentication modes.
- Avoid duplicate JWKS requests by implementing a thread-safe token validator instance.
