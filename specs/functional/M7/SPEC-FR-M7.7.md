# SPEC-FR-M7.7: Serve Welcome index.html in BFF

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.7                                |
| Status        | VERIFIED                                    |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M7.1                                |
| Supersedes    | none                                        |

## Context

The BFF (Backend For Frontend) currently provides API endpoints and developer utilities but lacks a user-friendly homepage at its root/index path. A premium welcome page served from the UI path (`/ui`) and `/ui/index.html` will provide a professional interface, verify the service is running, and link to essential developer tools.

## Specification

1. **Serving index.html**:
   - The BFF MUST serve the welcome page (`index.html`) at the HTTP paths `/ui` and `/ui/index.html` (configured dynamically via `bff.ui_path`).
   - The page content MUST be embedded inside the BFF binary using Go's `embed` package to maintain the stateless, self-contained container requirement of the service.

2. **Serving secure index.html (Secure Zone)**:
   - The BFF MUST serve a secure welcome page (`secure_index.html` located in `internal/bff/secure/index.html`) at the HTTP paths `/ui/secure/` and `/ui/secure/index.html` (configured dynamically under the UI path).
   - The page content MUST be embedded inside the BFF binary using Go's `embed` package.
   - Access to this path MUST be protected by a middleware (`AuthRedirectMiddleware`) that checks for a valid OIDC session cookie `bff_session_id`. If the session is missing, invalid, or expired, the user MUST be redirected to the OIDC login flow at `/api/v1/auth/login` (OIDC Webflow Binding).
   - The secure welcome page MUST display the welcome screen and provide a secure logout control/button.
   - The logout control MUST trigger a `POST` request to `/api/v1/auth/logout` using a standard HTML form submission.

3. **Visual Standards & Design**:
   - The UI for both index pages MUST be styled with premium, responsive Vanilla CSS.
   - It MUST feature a modern dark mode, glassmorphic elements, subtle micro-animations (e.g., button glow on hover, active state transitions), and premium typography.
   - It MUST include descriptive text welcoming the user to the Tacito Square BFF.

4. **Developer Navigation**:
   - The homepage MUST include links to helper routes: `/openapi.json`, `/healthz`, `/readyz`, and `/metrics`.

5. **SEO & Accessibility**:
   - The pages MUST include a proper title tag, a meta description, a single `<h1>` tag, and unique, descriptive IDs for all interactive elements to support browser automation.

## Acceptance Criteria

1. Requesting `http://localhost:8083/ui` or `http://localhost:8083/ui/` or `http://localhost:8083/ui/index.html` returns `200 OK` with `Content-Type: text/html; charset=utf-8` and renders the premium welcome page.
2. Requesting `http://localhost:8083/ui/secure/` or `http://localhost:8083/ui/secure/index.html` when NOT authenticated redirects (HTTP 302/307) to `/api/v1/auth/login`.
3. Requesting `http://localhost:8083/ui/secure/` or `http://localhost:8083/ui/secure/index.html` when authenticated returns `200 OK` and renders the secure welcome page.
4. The secure welcome page contains a logout form with `method="POST" action="/api/v1/auth/logout"` and an interactive submit button (`id="btn-logout"`).
5. Clicking the logout button terminates the session, clears the session cookie (scoped to the UI path), and redirects the user (either back to root `/ui` or OIDC end-session endpoint).
6. Pages conform to general web application development aesthetics (no placeholders, styled inputs, animations, Inter/Roboto fonts, dark mode).

## Test Plan

- **Automated Tests**:
  - Run tests in `bootstrap_test.go` asserting that `GET /ui` and `GET /ui/index.html` return `200 OK` with HTML content.
  - Run tests asserting that `GET /ui/secure/` redirects to `/api/v1/auth/login` when no cookie is present, and returns `200 OK` with secure content when a valid cookie is present.
- **Manual Verification**:
  - Start the BFF server and access `/ui/secure/` in a browser to check that it redirects to OIDC. After completing the login flow, verify it renders the secure index and clicking logout terminates the session correctly.

## Files Affected

- `internal/bff/bootstrap.go`
- `internal/bff/bootstrap_test.go`
- `internal/bff/index.html`
- `internal/bff/secure/index.html` [NEW]
- `internal/bff/adapters/inbound/http/session_middleware.go`
- `internal/bff/adapters/inbound/http/session_middleware_test.go`

