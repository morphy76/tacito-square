# SPEC-FR-M7.4: OIDC Login Flow (Keycloak)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.4                                |
| Status        | ACCEPTED                                    |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M7.1, SPEC-FR-M3.9                 |
| Supersedes    | none                                        |

## Context

Both the Configurator and Auditor UIs authenticate users via OIDC authorization code flow brokered by the BFF using Keycloak as the Identity Provider (IdP).

## Specification

### 1. Brokered Authorization Code Flow
* The BFF MUST act as the OIDC client. It MUST initiate the OIDC Authorization Code Flow with PKCE (`proof_key_for_code_exchange`) upon request.
* Redirect URIs MUST be handled by the BFF at `/api/v1/auth/callback`.

### 2. Session & User Profile Service
* **Session Cookie**: After exchanging the authorization code, the BFF MUST establish a secure cookie-based session in Redis (per SPEC-FR-M7.1) and return the session cookie to the browser.
* **UserInfo Endpoint**: The BFF MUST expose `GET /api/v1/auth/me`. This endpoint returns the user profile attributes, tenant IDs, and RBAC roles retrieved from Keycloak's UserInfo endpoint. The frontend UIs use this payload to drive role-based UI rendering.

### 3. Session Refresh and Lifecycle
* **Transparent Access Token Refresh**: When a UI API call is received and the cached access token in Redis is expired, the BFF MUST perform a refresh request to Keycloak using the cached refresh token. If Keycloak rejects the refresh, the BFF session MUST be destroyed, and a `401 Unauthorized` status returned.
* **Frontchannel Logout**: The UI logout MUST trigger a call to the BFF's logout endpoint, which invalidates the local session in Redis and redirects the user to Keycloak's OIDC end-session endpoint to terminate the SSO session.

### 4. Backchannel Logout Compliance
* The BFF MUST expose `POST /api/v1/auth/backchannel-logout`.
* Keycloak calls this endpoint when a logout occurs elsewhere in the SSO realm. The BFF MUST validate the signature of the incoming Logout Token (JWT) using Keycloak's JWKS and invalidate any matching Redis session keys immediately.

## Acceptance Criteria

1. **Token Isolation**: No raw OIDC tokens (Access, Refresh, or ID tokens) are ever sent to, or accessible by, the frontend UI application.
2. **Role Verification**: The UIs drive visual authorization checks based entirely on the JSON payload returned from `GET /api/v1/auth/me`.
3. **Session Invalidation**: Initiating backchannel logout immediately renders the browser's session cookie invalid on subsequent requests.
4. **Invalid Refresh Redirect**: An expired session that fails refresh results in a `401` HTTP response, prompting the UI to redirect the browser to the login flow.

## Test Plan

### Unit & Integration Tests
* **OIDC Redirect and Callback**: Assert proper redirection headers are generated for Keycloak login and that the callback handler successfully extracts the code.
* **User Profile Retrieval**: Test that `/api/v1/auth/me` returns the cached UserInfo correctly.
* **Backchannel Logout Signature**: Mock a Keycloak JWKS endpoint and send signed Logout Tokens (both valid and invalid) to test session eviction.
* **Token Expiration/Refresh**: Verify session eviction when the refresh token is expired or rejected by Keycloak.

## Files Affected

* `internal/bff/adapters/inbound/auth_handler.go` (Login, callback, logout, and `/me` routes)
* `internal/bff/adapters/outbound/oidc_client.go` (Keycloak OIDC client interface implementation)
* `internal/bff/application/service/auth_service.go` (OIDC state validation, session generation, token refresh logic)

