# BUG-M7.1: BFF OIDC Discovery Refused Connection due to localhost Issuer URL

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M7.1                                                           |
| Status        | IMPLEMENTED                                                        |
| Severity      | HIGH                                                               |
| Milestone     | M7 — BFF & UIs                                                     |
| Affects       | `internal/bff/adapters/outbound/oidc_http_client.go`, `cmd/bff/main.go` |
| Violates      | SPEC-FR-M7.4                                                       |
| Discovered    | Authentication callback flow error `http 500 failed to exchange authorization code: discovery request failed` |

## Problem Statement

In development environments where Keycloak is deployed inside a local Kubernetes cluster and exposed via a Traefik ingress on `localhost:80` (or `localhost:443`), the OIDC issuer URL is configured as `http://localhost/auth/realms/tacito` so that the user's browser redirects to it properly.

However, when the BFF container runs within the cluster:
1. It tries to verify/exchange authorization codes by making internal network requests to the OIDC provider's discovery URL at `http://localhost/auth/realms/tacito/.well-known/openid-configuration`.
2. Because `localhost` maps to the loopback interface (`127.0.0.1` or `[::1]`) inside the BFF pod, the connection is refused, yielding HTTP 500 error:
   `{"error":"failed to exchange authorization code: discovery request failed: Get \"http://localhost/auth/realms/tacito/.well-known/openid-configuration\": dial tcp [::1]:80: connect: connection refused"}`
3. Standard OIDC discovery returns endpoint fields pointing back to the public `localhost` host, meaning that subsequent calls to the token, userinfo, and JWKS endpoints will also fail if they resolve via `localhost`.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| BFF OIDC Client | `internal/bff/adapters/outbound/oidc_http_client.go` | Hardcoded reliance on `Issuer` for discovery and endpoints, with no option for internal service routing. |
| BFF Configuration | `cmd/bff/main.go` | Does not parse or configure an internal OIDC issuer fallback/alias. |
| Helm Chart Config | `tools/helm/tacito-square/templates/bff/deployment.yaml`, `values.yaml` | Does not expose an environment variable for mapping internal Keycloak service name. |
| Dev Environment Values | `tools/helm/dev-values.yaml` | Lacks setup of internal issuer target pointing to Keycloak K8s service. |

## Impact

The user cannot log in. The callback `/ui/api/v1/auth/callback` fails with 500 errors because the backend cannot contact Keycloak.

## Expected Behaviour

1. The BFF OIDC outbound client MUST support an optional configuration field `InternalIssuer`.
2. If `InternalIssuer` is set:
   - The initial OIDC discovery endpoint is queried using the `InternalIssuer` instead of the public `Issuer`.
   - The discovered endpoints (such as `token_endpoint`, `userinfo_endpoint`, and `jwks_uri`) are rewritten by replacing the public `Issuer` prefix with the `InternalIssuer` prefix.
   - Cryptographic verification of tokens continues to validate that the claims issuer equals the configured public `Issuer`.
3. In local development charts (`dev-values.yaml`), `internalIssuer` is configured to point to `http://ts-infra-keycloak-http/auth/realms/tacito`.

## Acceptance Criteria

1. A unit test verifying OIDC client discovery and token exchange successfully routes through a mock OIDC server via the `InternalIssuer` and redirects `localhost`-based endpoints to the mock server (RED Phase).
2. `OIDCHTTPClient` is updated to rewrite public endpoints to the internal issuer endpoint (GREEN Phase).
3. The bff builds and passes all unit tests.
4. Helm charts are updated to pass `TS_BFF_OIDC_INTERNAL_ISSUER` to the BFF deployment, with the dev values overriding it to point to the Keycloak service `http://ts-infra-keycloak-http/auth/realms/tacito`.
