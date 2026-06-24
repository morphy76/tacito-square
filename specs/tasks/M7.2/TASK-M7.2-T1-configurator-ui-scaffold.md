# TASK-M7.2-T1: Configurator UI Scaffolding & Monorepo Integration

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T1                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`)                               |
| Status      | DRAFT                                                              |
| Depends On  | SPEC-FR-M7.2                                                       |

## Objective

Scaffold a React 19 single-page application under `ui/configurator/` leveraging the React 19 Compiler, configured with local mock-server capabilities for development without running the full Go backend. Integrate the new frontend project into the root Makefile to support standard monorepo workflows (build, test, lint, clean). The UI will be built and deployed independently as a separate Kubernetes pod (Nginx static server), avoiding any need to rebuild or redeploy the Go BFF for frontend-only updates.

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
| `Makefile` | MODIFY |

## RED Phase

1. **Frontend Tests Setup**:
   - Implement a simple Vitest test suite in `ui/configurator/src/App.test.tsx` verifying that the app renders without crashing.
   - Run the frontend test command: `npm run test` (must fail/error because dependencies and source code do not exist yet).

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
             target: 'http://localhost:8083', // Proxies to local Go BFF
             changeOrigin: true
           }
         }
       }
     });
     ```
   - Build a standard SPA layout with premium, responsive Vanilla CSS (`index.css`) containing dark mode defaults, typography (Inter/Roboto), and simple layouts.
   - Create `mock-server/db.json` with initial schema/contracts (e.g., `/api/v1/auth/me` mock payload containing the `keeper-admin` role, and configurator mock data).
   - Configure scripts in `package.json` to enable local mock-driven development:
     - `"dev"`: Runs vite dev server proxying to local BFF (port 8083).
     - `"mock-server"`: Runs `json-server --watch mock-server/db.json --port 3001` (to be targeted by proxy in mock development mode).
     - `"dev:mock"`: Runs both concurrently (`concurrently "npm run mock-server" "vite --port 3000"` with proxy targets re-routed to port 3001).
   - Ensure the Vitest configuration is completed, and run `npm run test` to verify the frontend test passes.

2. **Update root Makefile**:
   - Define targets:
     - `build-ui-configurator`: Runs npm install and build inside `ui/configurator`.
     - `test-ui-configurator`: Runs Vitest tests.
     - `lint-ui-configurator`: Runs lint commands.
   - Inject these targets into standard monorepo workflows (`build`, `test`, `lint`, and `ci`).
   - Declare all targets as `.PHONY`.

## REFACTOR Phase

- Confirm React Compiler optimization works correctly by inspecting build logs.
- Verify node modules are correctly excluded in git tracking.
