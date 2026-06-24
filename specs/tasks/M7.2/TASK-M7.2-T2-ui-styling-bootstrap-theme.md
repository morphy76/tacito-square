# TASK-M7.2-T2: Shareable Bootstrap Theme & Index Pages Styling

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T2                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`), BFF (`internal/bff/`)        |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1                                         |

## Objective

Create a shareable custom Bootstrap theme that incorporates the premium Piazza Tacito aesthetics (dark mode, glassmorphism, steel/porphyry/water color palette, transitions, and animations) from `internal/bff/index.html` and `internal/bff/secure/index.html`. Apply this theme to the React-based Configurator UI application, and refactor the BFF index/welcome pages to reference this shareable theme instead of using inline custom CSS, aligning the styling across all user-facing interfaces.

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
   - Run tests (`make test`) or verify they fail to build/serve if the shareable theme is missing.
   - Add a unit test in `internal/bff/bootstrap_test.go` asserting that the BFF serves the shareable stylesheet `/ui/assets/bootstrap-theme.css` (e.g. `TestStaticUI_ServeBootstrapTheme` asserting `200 OK` and correct Content-Type).

2. **React UI Theme Verification (RED)**:
   - Add a frontend component test or visual regression check in `ui/configurator` asserting that Bootstrap variables are defined and the custom theme color properties (e.g. `--color-porphyry-glow`) are accessible in computed styles.
   - Run the frontend tests (`npm run test` inside `ui/configurator`) — must fail/error because the theme file doesn't exist yet and imports will fail.

## GREEN Phase

1. **Create the Custom Bootstrap Theme**:
   - Create a CSS (or SCSS) file `ui/shared/css/bootstrap-theme.css` embodying the design system defined in `internal/bff/index.html`.
   - Override standard Bootstrap parameters, including:
     - Background color (`#0b0c10`)
     - Foreground text colors, fonts (Outfit and Inter)
     - Component class overrides for Glassmorphism cards (`.glass-card`), ambient background components (`.ambient-bg`, `.pennone-mast`, `.zodiac-container`), and custom font weights.
   - Set up custom responsive gradients and transitions.

2. **Refactor BFF Welcome Index Pages**:
   - Modify `internal/bff/index.html` and `internal/bff/secure/index.html` to remove inline CSS blocks.
   - Add a stylesheet `<link>` referencing `/ui/assets/bootstrap-theme.css` to load the custom Bootstrap theme.
   - Use Bootstrap CSS classes (like `d-flex`, `justify-content-center`, `align-items-center`, grid columns, margins) for the overall layout.
   - Retain the animated header and the interactive SVG constellations.

3. **Update BFF static asset serving**:
   - Ensure the new shareable theme file `ui/shared/css/bootstrap-theme.css` is embedded in the BFF or copied/served via the BFF assets directory path under `/ui/assets/bootstrap-theme.css`.
   - Update `internal/bff/bootstrap.go` to serve this file under the assets path.
   - Run Go test suite: `make test` — all tests must pass.

4. **Integrate Theme in Configurator UI**:
   - In `ui/configurator/src/index.css`, import the shareable Bootstrap CSS theme file or compile it.
   - Update layout files or core layout components in the React 19 app to use standard Bootstrap elements and classes.
   - Run frontend tests: verify everything passes.

## REFACTOR Phase

- Audit stylesheet for duplicate declarations.
- Optimize CSS asset bundle sizes.
- Verify consistent rendering, dark mode, transitions, and responsive grid layouts on multiple viewport widths.
