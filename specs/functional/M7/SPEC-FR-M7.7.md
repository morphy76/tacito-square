# SPEC-FR-M7.7: Serve Welcome index.html in BFF

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.7                                |
| Status        | ACCEPTED                                    |
| Milestone     | M7                                          |
| Component     | bff                                         |
| Depends On    | SPEC-FR-M7.1                                |
| Supersedes    | none                                        |

## Context

The BFF (Backend For Frontend) currently provides API endpoints and developer utilities but lacks a user-friendly homepage at its root/index path. A premium welcome page served from the root path (`/`) and `/index.html` will provide a professional interface, verify the service is running, and link to essential developer tools.

## Specification

1. **Serving index.html**:
   - The BFF MUST serve the welcome page (`index.html`) at the HTTP paths `/` and `/index.html`.
   - The page content MUST be embedded inside the BFF binary using Go's `embed` package to maintain the stateless, self-contained container requirement of the service.

2. **Visual Standards & Design**:
   - The UI MUST be styled with premium, responsive Vanilla CSS.
   - It MUST feature a modern dark mode, glassmorphic elements, subtle micro-animations (e.g., button glow on hover, active state transitions), and premium typography.
   - It MUST include descriptive text welcoming the user to the Tacito Square BFF.

3. **Developer Navigation**:
   - The homepage MUST include links to helper routes: `/openapi.json`, `/healthz`, `/readyz`, and `/metrics`.

4. **SEO & Accessibility**:
   - The page MUST include a proper title tag, a meta description, a single `<h1>` tag, and unique, descriptive IDs for all interactive elements to support browser automation.

## Acceptance Criteria

1. Requesting `http://localhost:8083/index.html` or `http://localhost:8083/` returns `200 OK` with `Content-Type: text/html; charset=utf-8` and renders the premium welcome page.
2. The page is served from memory via embedded asset, requiring no local filesystem lookups at runtime.
3. The page includes navigation links with unique HTML IDs (e.g., `link-openapi`, `link-healthz`, `link-readyz`, `link-metrics`).
4. Page conforms to general web application development aesthetics (no placeholders, styled inputs, animations, Inter/Roboto fonts, dark mode).

## Test Plan

- **Automated Tests**:
  - Run tests in `bootstrap_test.go` asserting that `GET /` and `GET /index.html` return `200 OK` with HTML content.
- **Manual Verification**:
  - Start the BFF server and access the root route in a browser to inspect styling, animations, and links.

## Files Affected

- `internal/bff/bootstrap.go`
- `internal/bff/bootstrap_test.go`
- `internal/bff/index.html` [NEW]
