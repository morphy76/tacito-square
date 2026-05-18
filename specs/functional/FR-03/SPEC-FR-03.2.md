# SPEC-FR-03.2: Gin RBAC Middleware (JWT & Keycloak JWKS)

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-03.2                       |
| Status        | DRAFT                              |
| Milestone     | M3                                 |
| FR/NFR Ref    | FR-13.2                            |
| Component     | shared                             |
| Depends On    | SPEC-FR-13.1                       |

## Context

To secure the Keeper REST APIs, we need a Gin middleware capable of validating incoming Bearer JSON Web Tokens (JWT) against Keycloak's JSON Web Key Set (JWKS). Once validated, the middleware must extract the user's identity and roles to enforce RBAC rules and establish a reliable security context for downstream handlers.

## Specification

### 1. JWT Validation (`go-oidc`)

To avoid reinventing the wheel on standard OpenID Connect protocols, the middleware MUST use the `github.com/coreos/go-oidc/v3/oidc` library.

- **Initialization**: At startup, the application MUST create an `oidc.Provider` using Keycloak's discovery URL (`/realms/{realm}`). This automatically fetches and caches the JWKS endpoints.
- **Verification**: The middleware MUST extract the Bearer token from the `Authorization` header and verify it using `provider.Verifier(&oidc.Config{ClientID: expectedClientID})`.
- **Failure**: If the token is missing, malformed, expired, or cryptographically invalid, the middleware MUST immediately abort the request and return a `401 Unauthorized` JSON response.

### 2. Claim Extraction & Role Normalization

Keycloak can place roles in two different payload claims depending on configuration. The middleware MUST safely parse both to support varying deployment patterns:

- `realm_access.roles`: An array of strings representing global realm roles.
- `resource_access.{client_id}.roles`: An array of strings representing client-specific roles.

The middleware MUST extract the `sub` claim (representing the actor's UUID) and merge the roles from both `realm_access` and `resource_access` into a single distinct array of strings.

### 3. Context Injection (`Principal`)

Once verified and extracted, the identity MUST be injected into the request context to avoid duplicate parsing downstream.

- **Domain Struct**: The middleware MUST instantiate a strongly-typed `Principal` struct:
  ```go
  type Principal struct {
      Subject string
      Roles   []string
  }
  ```
- **Injection**: The `Principal` instance MUST be saved into the `*gin.Context` (e.g., `c.Set("principal", principal)`).
- **Propagation**: Because domain services accept `context.Context`, the middleware (or a utility function) MUST also provide a way to inject this `Principal` into the Go `context.Context` payload so it can be retrieved by layers unaware of the Gin framework.

## Acceptance Criteria

1. Requests with a missing or invalid `Authorization` header receive a `401 Unauthorized`.
2. The `go-oidc` verifier successfully caches the Keycloak JWKS and automatically rotates keys without application restarts.
3. Tokens containing roles in `realm_access` are successfully parsed and attached to the Principal.
4. Tokens containing roles in `resource_access` are successfully parsed and attached to the Principal.
5. Downstream handlers can successfully retrieve the strongly-typed `Principal` struct from the Gin context using a type-safe getter (e.g., `GetPrincipal(c) (*Principal, error)`).
