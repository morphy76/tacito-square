# TASK-M7.2-T1: Configurator UI Scaffolding & BFF Static Integration

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T1                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`), BFF (`internal/bff/`)        |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.2, SPEC-FR-M7.7                                         |

## Objective

Scaffold a React 19 single-page application under `ui/configurator/` leveraging the React 19 Compiler, configured with local mock-server capabilities for development without running the full backend stack. Update the Go BFF to serve the built static assets from a configurable directory with high-performance HTTP caching headers (aggressive caching for versioned assets, conditional requests/ETags for non-versioned files and index.html). Finally, update the BFF Docker build pipeline and root Makefile.

## Files

| File | Action |
|------|--------|
| `ui/configurator/package.json` | NEW |
| `ui/configurator/vite.config.ts` | NEW |
| `ui/configurator/tsconfig.json` | NEW |
| `ui/configurator/index.html` | NEW |
| `ui/configurator/src/main.tsx` | NEW |
| `ui/configurator/src/App.tsx` | NEW |
| `ui/configurator/src/index.css` | NEW |
| `ui/configurator/mock-server/db.json` | NEW |
| `internal/bff/bootstrap.go` | MODIFY |
| `internal/bff/bootstrap_test.go` | MODIFY |
| `cmd/bff/main.go` | MODIFY |
| `tools/docker/Dockerfile.bff` | MODIFY |
| `Makefile` | MODIFY |

## RED Phase

1. **Frontend Tests setup**:
   - Implement a simple Vitest test suite in `ui/configurator/src/App.test.tsx` verifying that the app renders without crashing.
   - Run the frontend test command: `npm run test` (must fail/error because dependencies and source code do not exist yet).

2. **BFF Routing & Cache Tests**:
   - Add unit tests in `internal/bff/bootstrap_test.go` to assert file serving behaviors when `TS_BFF_UI_ASSETS_DIR` is set to a mock directory:
     - `TestStaticUI_CacheHeaders_HashedAsset`: Assert that requests for a hashed asset (e.g. `/ui/assets/index-B1a89c2.js`) return `200 OK`, `Cache-Control: public, max-age=31536000, immutable`, and matching file contents.
     - `TestStaticUI_CacheHeaders_IndexHTML`: Assert that requests for `/ui/index.html` return `200 OK`, `Cache-Control: public, max-age=0, must-revalidate`, and a strong double-quoted `ETag`.
     - `TestStaticUI_ConditionalRequest`: Send a request for `/ui/index.html` with `If-None-Match` set to the previously computed ETag, and assert it returns `304 Not Modified` with an empty response body.
     - `TestStaticUI_SPARoutingFallback`: Assert that requests to non-existent assets (e.g. `/ui/agents` or `/ui/communities`) fall back to serving `index.html` with `200 OK` and caching headers, enabling client-side routing.
   - Run Go test suite: `make test` — must fail to compile or fail test execution because BFF doesn't support the configuration or the serving logic yet.

## GREEN Phase

1. **Scaffold React 19 Frontend (`ui/configurator/`)**:
   - Create `package.json` with React 19 (`react@19.0.0`, `react-dom@19.0.0`) and devDependencies like `vite`, `@vitejs/plugin-react`, `babel-plugin-react-compiler`, `typescript`, `concurrently`, `json-server`, `vitest`, `@testing-library/react`.
   - Setup React 19 Compiler in `vite.config.ts`:
     ```typescript
     import { defineConfig } from 'vite';
     import react from '@vitejs/plugin-react';

     export default defineConfig({
       plugins: [
         react({
           babel: {
             plugins: [
               ['babel-plugin-react-compiler', {}]
             ]
           }
         })
       ],
       server: {
         port: 3000,
         proxy: {
           '/api/v1': {
             target: 'http://localhost:8083', // Proxies to Go BFF or mock-server
             changeOrigin: true
           }
         }
       }
     });
     ```
   - Build a standard SPA layout with premium, responsive Vanilla CSS (`index.css`) containing dark mode defaults, typography (Inter/Roboto), and simple layouts.
   - Create `mock-server/db.json` with initial schema/contracts (e.g., `/api/v1/auth/me` mock payload containing the `keeper-admin` role, and configurator mock data).
   - Configure scripts in `package.json` to enable local mock-driven development:
     - `"dev"`: Runs vite dev server proxying to local BFF.
     - `"mock-server"`: Runs `json-server --watch mock-server/db.json --port 3001` (to be targeted by proxy in mock development mode).
     - `"dev:mock"`: Runs both concurrently (`concurrently "npm run mock-server" "vite --port 3000"` with proxy targets re-routed to port 3001).
   - Ensure the Vitest configuration is completed, and run `npm run test` to verify the frontend test passes.

2. **Implement BFF Config & Static Serving**:
   - In `cmd/bff/main.go` and `internal/bff/bootstrap.go`, add `bff.ui_assets_dir` / `TS_BFF_UI_ASSETS_DIR` to Viper configuration, default empty.
   - In `internal/bff/bootstrap.go`, implement directory-based asset serving under `/ui` (or `cfg.UIPath`) if `cfg.UIAssetsDir` is set:
     - Resolve the requested file path relative to the configured UI assets directory, cleaning and verifying that the path does not escape the directory root.
     - If the file exists on disk:
       - For hashed files (e.g., under `/assets/`), set header `Cache-Control: public, max-age=31536000, immutable`.
       - For non-hashed files (e.g., `index.html`), calculate/cache a strong `ETag` (e.g. SHA-256 of contents), set `Cache-Control: public, max-age=0, must-revalidate`, and process `If-None-Match` headers, returning `304 Not Modified` when applicable.
       - Stream the file using Gin's context.
     - If the file does not exist, assume Single Page Application (SPA) client-side routing, and fall back to serving `index.html` from the assets directory using the same caching/ETag logic.
   - Run Go test suite: `make test` — all tests must pass.

3. **Update root Makefile**:
   - Define targets:
     - `build-ui-configurator`: Runs npm install and build inside `ui/configurator`.
     - `test-ui-configurator`: Runs Vitest tests.
     - `lint-ui-configurator`: Runs lint commands.
   - Inject these targets into standard monorepo workflows (`build`, `test`, `lint`, and `ci`).
   - Declare all targets as `.PHONY`.

4. **Update Docker Build**:
   - Modify `tools/docker/Dockerfile.bff` to build the UI assets using a Node.js stage (`node:22-alpine`) first.
   - Copy the built assets from the UI stage into the final runner image (`gcr.io/distroless/base-nossl-debian13:nonroot`).
   - Set environment variable `TS_BFF_UI_ASSETS_DIR=/app/ui-configurator` inside the image so that the BFF serves them automatically in deployment.

## REFACTOR Phase

- Ensure there are no path traversal vulnerabilities in the BFF static asset resolver (e.g. verify `filepath.Clean` and check `strings.HasPrefix`).
- Confirm the Go binary size remains lightweight and does not embed the large React application folder.
- Confirm React Compiler optimization works correctly by inspecting build logs.
