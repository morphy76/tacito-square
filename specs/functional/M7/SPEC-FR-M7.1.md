# SPEC-FR-M7.1: BFF API Bridge Layer

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.1                                |
| Status        | VERIFIED                                    |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M2.6                                |
| Supersedes    | none                                        |

## Context

The BFF (Backend For Frontend) serves as the sole entry point and API bridge layer for the user interfaces. It decouples the UI from concrete backend microservice layouts (e.g., Keeper), aggregates data optimized for frontend views, manages OIDC authentication sessions, and handles real-time event streaming via NATS.

## Specification

### 1. Route Namespace & View-Model Bridge
* The BFF MUST serve its UI-facing APIs under the dedicated route namespace `/api/v1/` (e.g., `/api/v1/configurator/*`, `/api/v1/auditor/*`).
* The BFF MUST NOT act as a dumb pass-through proxy. It MUST aggregate and translate backend data models (optimized for business logic) into structured view-models tailored for UI rendering.

### 2. Token Handler Pattern & Session Management
* **Stateless Session Storage**: The BFF MUST use **Redis** to store OIDC access tokens, refresh tokens, and cached UserInfo payloads, keyed by a cryptographically secure random Session ID.
* **Secure Cookie Binding**: The Session ID MUST be sent to the client browser in an encrypted, `HttpOnly`, `Secure`, `SameSite=Strict` cookie.
* **Transparent Token Refresh**: If an incoming UI request presents a valid session cookie but the access token cached in Redis is expired, the BFF MUST transparently use the cached refresh token to request a new access token from Keycloak, update the cache in Redis, and complete the request.
* **SSO Backchannel Logout**: The BFF MUST expose an OIDC-compliant backchannel logout endpoint `/api/v1/auth/backchannel-logout`. Upon receiving a valid Logout Token from the IAM (Keycloak), the BFF MUST invalidate all matching user sessions in Redis.

### 3. Tenant Extraction & Internal Propagation
* **Opaque Token Resolution**: The BFF MUST fetch user profile details from the OIDC UserInfo endpoint upon authentication and cache the payload in Redis.
* **Tenant Enrichment**: The BFF MUST retrieve the tenant identifier (e.g., `tenantid`, `subscriptionid`) from the UserInfo profile and enrich the Go `context.Context` for tracing, structured logging, and metrics.
* **Downstream Propagation**: Outbound calls to backend microservices (e.g., Keeper REST endpoints) MUST propagate the original `Authorization: Bearer <opaque_token>` header. The backend services will perform their own tenant resolution using the shared code module.

### 4. Real-time Event Streaming (SSE Gateway)
* The BFF MUST serve an SSE endpoint (e.g., `/api/v1/events/stream`) that forwards real-time event feeds from backend APIs to the UI.
* The BFF MUST NOT connect directly to backend message brokers (e.g., NATS). Instead, it MUST establish secure proxy connections to downstream backend HTTP SSE endpoints, forward the stream payloads to the UI, and handle connection lifecycles and authentication.

### 5. Architectural Standards
* The BFF MUST be built using **Go 1.26** and **Gin** (per SPEC-NFR-STACK and SPEC-NFR-HTTP).
* The BFF MUST implement **Hexagonal Architecture**: the core orchestrators must interact with backends through outbound port interfaces (e.g., `ports.KeeperClient`, `ports.BackendEventClient`).
* Outbound calls to external APIs or other microservices MUST use tested circuit breakers and timeouts (per SPEC-NFR-CLOUD).

## Acceptance Criteria

1. **View-Model Isolation**: No frontend UI communicates directly with Keeper or other backend services; all requests route through the BFF under `/api/v1/`.
2. **Cookie Security**: Zero tokens (Access, Refresh, ID) are exposed to client-side Javascript. All cookies are marked `HttpOnly` and `Secure`.
3. **Session Invalidation**: Invoking the Backchannel Logout endpoint destroys the active Redis session, and the next UI request returns a `401 Unauthorized` response.
4. **Tenant Scope**: All internal logs and traces in the BFF contain the resolved `tenant_id` field derived from UserInfo.
5. **Autoscaling & Statelessness**: BFF pods scale horizontally using standard HPA templates without session stickiness or memory requirements, relying entirely on the Redis session cluster.

## Test Plan

### Unit & Integration Tests
* **Auth Session Handlers**: Test OIDC authentication callback, cookie creation, and session storage in a mock Redis client.
* **Token Refresh Interceptor**: Mock Keycloak's token endpoint to verify transparent access token refresh on expired cached tokens.
* **Backchannel Logout**: Send a mock signed Logout Token to the backchannel logout endpoint and assert that the corresponding session is deleted from Redis.
* **Tenant Resolution**: Verify that calling the UserInfo endpoint populates the context with the tenant identity.
* **SSE Event Stream**: Establish an SSE connection to the BFF, trigger mock events on a simulated backend HTTP SSE stream, and verify the BFF successfully forwards the events.

### Contract Tests
* Validate that downstream calls to Keeper adhere to `api/openapi/openapi.json`.

## Files Affected

* `internal/bff/domain/` (BFF domain aggregates, view-models, session entities)
* `internal/bff/application/ports/` (Ports for OIDC provider, Session Store, Keeper client, backend SSE clients)
* `internal/bff/application/service/` (Core use cases: session management, view orchestration, event bridge)
* `internal/bff/adapters/inbound/` (Gin HTTP handlers, SSE gateway handler, backchannel logout handler)
* `internal/bff/adapters/outbound/` (Redis session repository, OIDC HTTP client, Keeper HTTP client, Backend SSE proxy client)
* `deploy/helm/tacito-square/templates/bff/` (BFF deployment, HPA, and configuration charts)

