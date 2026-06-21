# TASK-M7.1-T3: BFF Application Services (`internal/bff/application/service/`)

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Task ID     | TASK-M7.1-T3                                       |
| Spec        | SPEC-FR-M7.1                                       |
| Boundary    | BFF Application Services — `internal/bff/application/service/` |
| Status      | IMPLEMENTED                                        |
| Depends On  | TASK-M7.1-T2                                       |

## Objective

Implement the BFF use case orchestrators: `SessionService` (OIDC authentication lifecycle, transparent token refresh, backchannel logout) and `EventBridgeService` (SSE stream proxying). Both services depend exclusively on outbound port interfaces — no concrete infrastructure clients.

## Files

| File | Action |
|------|--------|
| `internal/bff/application/service/session_service.go` | NEW |
| `internal/bff/application/service/session_service_test.go` | NEW |
| `internal/bff/application/service/event_bridge_service.go` | NEW |
| `internal/bff/application/service/event_bridge_service_test.go` | NEW |

## RED Phase

Write unit tests using mock implementations of the outbound ports (using `testify/mock` or hand-rolled mocks):

**`session_service_test.go`**:

- `TestSessionService_HandleCallback_Success`: Mock `OIDCProvider.ExchangeCode`, `FetchUserInfo`, and `SessionStore.Save`. Assert that a valid auth code creates a session with the correct `TenantID` populated from `UserInfoPayload` and stores it in Redis.
- `TestSessionService_HandleCallback_ExchangeError`: Assert that an `ExchangeCode` failure propagates as an error (no session is stored).
- `TestSessionService_RefreshSession_ExpiredToken`: Construct a session with expired `AccessTokenExpiresAt`, mock `OIDCProvider.RefreshToken` returning a new token set, and assert the session is updated in the store.
- `TestSessionService_RefreshSession_ValidToken`: Assert that if the cached token is still valid, `RefreshToken` is NOT called (no unnecessary refresh).
- `TestSessionService_BackchannelLogout_Success`: Mock `OIDCProvider.ValidateLogoutToken` returning `(sub, "", nil)`, mock `SessionStore.DeleteByUserID`. Assert the store `DeleteByUserID` is called with the validated `sub`.
- `TestSessionService_Logout_DeletesSession`: Mock `SessionStore.Delete` and assert it is called with the correct session ID.

**`event_bridge_service_test.go`**:

- `TestEventBridgeService_StreamEvents_ForwardsChannel`: Mock `BackendEventClient.StreamEvents` returning a channel. Assert the bridge service passes the same channel through.
- `TestEventBridgeService_StreamEvents_ContextCancellation`: Cancel the context after opening the stream; assert the underlying `BackendEventClient` channel is drained and the returned channel is closed.

Run `make test` — must fail (RED).

## GREEN Phase

1. Create `internal/bff/application/service/session_service.go`:
   - Define `SessionService` struct with constructor `NewSessionService(store outbound.SessionStore, oidc outbound.OIDCProvider, cfg SessionConfig) *SessionService`.
   - `SessionConfig` holds OIDC client ID/secret, redirect URI, session TTL, and Keycloak issuer URL — sourced via Viper at wire-up.
   - Implement `HandleCallback`: call `ExchangeCode → FetchUserInfo → NewSession → store.Save`.
   - Implement `RefreshSession`: call `store.Get`; if `session.IsExpired()`, call `oidc.RefreshToken`, update session fields, call `store.Save`.
   - Implement `BackchannelLogout`: call `oidc.ValidateLogoutToken` to extract `sub`; call `store.DeleteByUserID(ctx, sub)`.
   - Implement `Logout`: call `store.Delete(ctx, sessionID)`.
   - Implement `GetSession`: call `store.Get`; return `ErrSessionExpired` if `session.IsExpired()` without attempting refresh.
   - Enrich Go `context.Context` with `tenantID` and `userID` using a shared context-key constant (to be consumed by middleware for logging and tracing).

2. Create `internal/bff/application/service/event_bridge_service.go`:
   - Define `EventBridgeService` struct with constructor `NewEventBridgeService(client outbound.BackendEventClient) *EventBridgeService`.
   - Implement `StreamEvents(ctx, tenantID)`: delegate directly to `client.StreamEvents(ctx, tenantID)`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm `SessionService` and `EventBridgeService` import **zero** packages from `adapters/` or any external infrastructure library.
- Verify `RefreshSession` uses a mutex or relies on Redis atomicity (e.g., using Redis `SET NX`) to prevent a thundering-herd of concurrent refreshes for the same session.
- Verify all goroutines spawned within services are governed by the passed `context.Context`.
- Confirm zerolog structured logging is used for key transitions: session created, session refreshed, session invalidated, backchannel logout executed.
