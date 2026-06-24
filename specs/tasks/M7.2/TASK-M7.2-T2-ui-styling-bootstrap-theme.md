# TASK-M7.2-T2: Shareable Bootstrap Theme & Index Pages Styling

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T2                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`), BFF (`internal/bff/`)        |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1                                         |

## Objective

Create a shareable custom Bootstrap theme that incorporates the premium Piazza Tacito aesthetics (dark mode, glassmorphism, steel/porphyry/water color palette, transitions, and animations). Apply this theme to the React-based Configurator UI, and refactor the BFF index/welcome pages to reference this theme. To keep the BFF completely decoupled and stateless, use Go `go:embed` to bundle the shareable theme stylesheet and welcome HTML files directly into the BFF binary.

## Files

| File | Action |
|------|--------|
| `ui/shared/css/bootstrap-theme.css` | NEW |
| `internal/bff/index.html` | MODIFY |
| `internal/bff/secure/index.html` | MODIFY |
| `ui/configurator/src/index.css` | MODIFY |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |

## RED Phase

1. **BFF Index Pages Styling Verification (RED)**:
   - Add a test or verify that index pages do not contain a large `<style>` block with custom hardcoded inline styles.
   - Add a unit test in `internal/bff/bootstrap_test.go` asserting that the BFF serves the shareable stylesheet `/ui/assets/bootstrap-theme.css` (e.g. `TestStaticUI_ServeBootstrapTheme` asserting `200 OK` and Content-Type `text/css` with the embedded stylesheet content).
   - Run tests (`make test`) — must fail to compile or execute since the theme file is missing.

2. **React UI Theme Verification (RED)**:
   - Add a frontend component test or visual regression check in `ui/configurator` asserting that Bootstrap variables are defined and the custom theme color properties (e.g. `--color-porphyry-glow`) are accessible in computed styles.
   - Run the frontend tests (`npm run test` inside `ui/configurator`) — must fail/error because the theme file doesn't exist yet and imports will fail.

## GREEN Phase

1. **Create the Custom Bootstrap Theme**:
   - Create a CSS file `ui/shared/css/bootstrap-theme.css` embodying the design system.
   - Override standard Bootstrap parameters, including:
     - Background color (`#0b0c10`)
     - Foreground text colors, fonts (Outfit and Inter)
     - Component class overrides for Glassmorphism cards (`.glass-card`), ambient background components (`.ambient-bg`, `.pennone-mast`, `.zodiac-container`), and custom font weights.
   - Set up custom responsive gradients and transitions.

2. **Refactor BFF Welcome Index Pages**:
   - Modify `internal/bff/index.html` and `internal/bff/secure/index.html` to remove inline CSS blocks.
   - Add a stylesheet `<link>` referencing `/ui/assets/bootstrap-theme.css` to load the custom Bootstrap theme.
   - Use Bootstrap CSS classes (like `d-flex`, `justify-content-center`, `align-items-center`, grid columns, margins) for the overall layout.

3. **Implement BFF Embed & Static Serving**:
   - In `internal/bff/bootstrap.go`, use `//go:embed` directives to embed:
     - `internal/bff/index.html`
     - `internal/bff/secure/index.html`
     - `../../ui/shared/css/bootstrap-theme.css` (using relative path or symlink if required, or mapping via a shared package structure)
   - Implement route handlers in Gin:
     - `GET /` serves the embedded `index.html`.
     - `GET /secure` serves the embedded `secure/index.html` (gated by session middleware).
     - `GET /ui/assets/bootstrap-theme.css` serves the embedded stylesheet content with correct `text/css` headers.
   - Run Go test suite: `make test` — all tests must pass.

4. **Integrate Theme in Configurator UI**:
   - In `ui/configurator/src/index.css`, import the shareable Bootstrap CSS theme file.
   - Update layout files or core layout components in the React 19 app to use standard Bootstrap elements and classes.
   - Run frontend tests: verify everything passes.

## REFACTOR Phase

- Audit stylesheet for duplicate declarations.
- Verify consistent rendering, dark mode, transitions, and responsive layouts on multiple viewport widths.
