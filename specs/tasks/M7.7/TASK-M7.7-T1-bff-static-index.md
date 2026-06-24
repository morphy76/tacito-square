# TASK-M7.7-T1: BFF Static Welcome Page (`internal/bff/`)

| Field       | Value                                              |
|-------------|----------------------------------------------------|
| Task ID     | TASK-M7.7-T1                                       |
| Spec        | SPEC-FR-M7.7                                       |
| Boundary    | BFF Welcome Page — `internal/bff/`                 |
| Status      | VERIFIED                                           |
| Depends On  | none                                               |

## Objective

Create a premium welcome page `index.html` and serve it at `/` and `/index.html` using Go's `embed` package in the BFF component.

## Files

| File | Action |
|------|--------|
| `internal/bff/index.html` | NEW |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |

## RED Phase

Update `internal/bff/bootstrap_test.go` to add tests for `/` and `/index.html`:
- Ensure `GET /` returns `200 OK` with `Content-Type: text/html` and contains standard greeting text.
- Ensure `GET /index.html` returns `200 OK` with `Content-Type: text/html` and contains standard greeting text.

Run tests to verify that they fail (RED).

## GREEN Phase

1. **Create `internal/bff/index.html`**:
   - Create a premium welcome page with modern design (dark mode, glassmorphism, glowing accents, smooth hover animations).
   - Use Google Fonts (e.g., Inter) for typography.
   - Include a single `<h1>` saying "Hello from Tacito Square BFF!".
   - Add links to `/openapi.json`, `/healthz`, `/readyz`, and `/metrics` with unique HTML IDs (`link-openapi`, `link-healthz`, `link-readyz`, `link-metrics`).
2. **Update `internal/bff/bootstrap.go`**:
   - Embed `index.html` using `//go:embed index.html`.
   - Register route GET `/index.html` and GET `/` to serve the embedded welcome page.

Run tests to verify that they pass (GREEN).

## REFACTOR Phase

- Ensure that CSS is clean and self-contained in `index.html`.
- Double-check that we are serving with correct content-type header.
