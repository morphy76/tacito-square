# TASK-M7.1-T5: BFF Inbound Adapters / HTTP Handlers (`internal/bff/adapters/inbound/`)

| Field       | Value                                                 |
|-------------|-------------------------------------------------------|
| Task ID     | TASK-M7.1-T5                                          |
| Spec        | SPEC-FR-M7.1                                          |
| Boundary    | BFF Inbound Adapters — `internal/bff/adapters/inbound/http/` |
| Status      | VERIFIED                                               |
| Depends On  | TASK-M7.1-T3                                          |

## Objective

Implement all Gin HTTP handler groups under the `/api/bff/v1/` namespace: the OIDC authentication flow handlers (login initiation, callback, logout), the OIDC backchannel logout handler, and the SSE gateway handler. Include the session middleware that authenticates every protected route.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/auth_handlers.go` | NEW |
| `internal/bff/adapters/inbound/http/auth_handlers_test.go` | NEW |
| `internal/bff/adapters/inbound/http/sse_handler.go` | NEW |
| `internal/bff/adapters/inbound/http/sse_handler_test.go` | NEW |
| `internal/bff/adapters/inbound/http/session_middleware.go` | NEW |
| `internal/bff/adapters/inbound/http/session_middleware_test.go` | NEW |
| `internal/bff/adapters/inbound/http/routes.go` | NEW |

## RED Phase

Write Gin test-mode handler tests using `httptest.NewRecorder()` and mock `inbound.SessionUseCase` / `inbound.EventStreamUseCase`:

**`auth_handlers_test.go`**:
- `TestAuthHandler_Login_RedirectsToOIDC`: Call `GET /api/bff/v1/auth/login`; assert the response is `302 Found` with a `Location` header pointing to the OIDC provider.
- `TestAuthHandler_Callback_Success_SetsCookie`: Mock `SessionUseCase.HandleCallback` returning a valid session; call `GET /api/bff/v1/auth/callback?code=abc&state=xyz`; assert response is a redirect and a `Set-Cookie` header is present with `HttpOnly`, `Secure`, and `SameSite=Strict` attributes.
- `TestAuthHandler_Callback_ExchangeFailure`: Mock `HandleCallback` returning an error; assert `500 Internal Server Error` JSON error response.
- `TestAuthHandler_Logout_ClearsCookie`: Call `POST /api/bff/v1/auth/logout` with a valid session cookie; mock `SessionUseCase.Logout`; assert session cookie is cleared (expired) in the response.
- `TestAuthHandler_BackchannelLogout_Success`: POST a mock raw logout token to `/api/bff/v1/auth/backchannel-logout`; mock `SessionUseCase.BackchannelLogout` returning `nil`; assert `200 OK`.
- `TestAuthHandler_BackchannelLogout_InvalidToken`: Mock `BackchannelLogout` returning an error; assert `400 Bad Request`.

**`sse_handler_test.go`**:
- `TestSSEHandler_StreamEvents_ForwardsEvents`: Inject a mock `EventStreamUseCase` returning a buffered channel with 3 events; call `GET /api/bff/v1/events/stream`; assert `Content-Type: text/event-stream`, and all 3 event payloads appear in the response body as SSE frames.
- `TestSSEHandler_StreamEvents_RequiresAuth`: Call the SSE endpoint without a session cookie; assert `401 Unauthorized`.

**`session_middleware_test.go`**:
- `TestSessionMiddleware_ValidCookie_PopulatesContext`: Provide a valid session cookie; mock `SessionUseCase.GetSession` returning a non-expired session; assert the handler receives `tenantID` in context.
- `TestSessionMiddleware_MissingCookie_Returns401`: Omit cookie; assert `401 Unauthorized`.
- `TestSessionMiddleware_ExpiredSession_AttemptRefresh`: Mock `GetSession` returning `ErrSessionExpired`; mock `RefreshSession` returning a refreshed session; assert the request proceeds with the new session.
- `TestSessionMiddleware_RefreshFails_Returns401`: Mock both `GetSession` and `RefreshSession` failing; assert `401 Unauthorized` and cookie is cleared.

Run `make test` — must fail (RED).

## GREEN Phase

1. **`session_middleware.go`**:
   - Read the `bff_session_id` cookie value.
   - Call `SessionUseCase.GetSession(ctx, sessionID)`.
   - If `ErrSessionExpired`, call `SessionUseCase.RefreshSession`; on failure, clear the cookie and abort with `401`.
   - Enrich `gin.Context` with `tenantID`, `userID`, and OTel span attributes.
   - Inject resolved `tenantID` into the zerolog context for structured logging.

2. **`auth_handlers.go`**:
   - `LoginHandler`: Call `SessionUseCase.InitiateLogin`; redirect to the returned auth URL; store OIDC `state` in a short-lived, secure cookie.
   - `CallbackHandler`: Read `code` and `state` from query params; validate `state` cookie; call `HandleCallback`; write encrypted session cookie (`HttpOnly`, `Secure`, `SameSite=Strict`); redirect to UI root.
   - `LogoutHandler`: Read session cookie; call `SessionUseCase.Logout`; expire/clear the cookie; redirect to OIDC end-session endpoint.
   - `BackchannelLogoutHandler`: Read `logout_token` from POST form body (per OIDC backchannel spec); call `SessionUseCase.BackchannelLogout`; return `200 OK` or `400 Bad Request`.

3. **`sse_handler.go`**:
   - Extract `tenantID` from the Gin context (set by session middleware).
   - Call `EventStreamUseCase.StreamEvents(ctx, tenantID)`.
   - Set response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`.
   - Write events from the channel as SSE frames (`data: <payload>\n\n`) using `c.Writer.Flush()` after each frame.
   - Send a heartbeat comment (`: keep-alive\n\n`) every `bff.sse.heartbeat_seconds` seconds (default: 30).
   - Terminate on context cancellation or channel close.

4. **`routes.go`**:
   - Implement `RegisterRoutes(r *gin.Engine, sessionUC inbound.SessionUseCase, eventUC inbound.EventStreamUseCase)`.
   - Register public routes: `GET /api/bff/v1/auth/login`, `GET /api/bff/v1/auth/callback`, `POST /api/bff/v1/auth/backchannel-logout`.
   - Register session-protected routes under a group using `SessionMiddleware`: `POST /api/bff/v1/auth/logout`, `GET /api/bff/v1/events/stream`.
   - Register placeholder groups for future specs: `/api/bff/v1/configurator/` and `/api/bff/v1/auditor/`.

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm the session cookie `Name` is `bff_session_id`, and all cookie attributes (`HttpOnly: true`, `Secure: true`, `SameSite: http.SameSiteStrictMode`) are always set — never conditional.
- Verify unified error JSON is used: `{"error": "message"}` (per SPEC-NFR-HTTP).
- Verify no token values appear in any log line at any level.
- Confirm OTel traceparent is extracted from inbound requests by Gin middleware and propagated to the context.
