# TASK-M7.7-T2: BFF Secure Index & OIDC Webflow Binding (`internal/bff/`)

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Task ID     | TASK-M7.7-T2                                       |
| Spec        | SPEC-FR-M7.7                                       |
| Boundary    | BFF Secure Zone — `internal/bff/`                  |
| Status      | VERIFIED                                           |
| Depends On  | TASK-M7.7-T1                                       |

## Objective

Create a secure version of the welcome page at `internal/bff/secure/index.html` with a logout form, configure the `AuthRedirectMiddleware` to dynamically redirect unauthenticated users to `/api/v1/auth/login`, register routes `/ui/secure/` and `/ui/secure/index.html`, and ensure they serve the secure homepage.

## Files

| File | Action |
|------|--------|
| `internal/bff/adapters/inbound/http/session_middleware.go` | MODIFY |
| `internal/bff/adapters/inbound/http/session_middleware_test.go` | MODIFY |
| `internal/bff/secure/index.html` | NEW |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |

## RED Phase

1. Add tests in `internal/bff/adapters/inbound/http/session_middleware_test.go` for `AuthRedirectMiddleware`:
   - `TestAuthRedirectMiddleware_ValidCookie_Continues`: Valid cookie sets context attributes and yields.
   - `TestAuthRedirectMiddleware_MissingCookie_Redirects`: Missing cookie redirects to `/api/v1/auth/login`.
   - `TestAuthRedirectMiddleware_ExpiredSession_Redirects`: If refresh fails, clear cookie and redirect to login.
2. Add tests in `internal/bff/bootstrap_test.go` for `/ui/secure/` and `/ui/secure/index.html`:
   - `TestBFFServer_SecureIndex_NoCookie_Redirects`: Ensure unauthenticated access redirects with HTTP 302 to `/api/v1/auth/login`.
   - `TestBFFServer_SecureIndex_WithCookie_Success`: Ensure authenticated access returns HTTP 200 and the secure welcome page.

Run tests to verify they fail (RED).

## GREEN Phase

1. **Update `internal/bff/adapters/inbound/http/session_middleware.go`**:
   - Implement `AuthRedirectMiddleware(sessionUC inbound.SessionUseCase)` which redirects to `/api/v1/auth/login` on failure/expiration.
2. **Create `internal/bff/secure/index.html`**:
   - Replicate the premium Piazza Tacito layout.
   - Add a POST form targeting `/api/v1/auth/logout` containing a submit button `id="btn-logout"` to trigger logout.
3. **Update `internal/bff/bootstrap.go`**:
   - Embed `secure/index.html` using `//go:embed secure/index.html` as `secureIndexHTML`.
   - Add the `/secure` router group using `AuthRedirectMiddleware` (exported from http adapters) and bind `/` and `/index.html` inside it to return `secureIndexHTML`.

Run tests to verify they pass (GREEN).

## REFACTOR Phase

- Clean up code, verify no duplicate code between middleware layers.
- Check responsive styles on the secure logout page.
