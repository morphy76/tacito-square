# TASK-M7.2-T3: Configurator UI Auth Guard & Role-Based Access Control

| Field       | Value                                                              |
|-------------|--------------------------------------------------------------------|
| Task ID     | TASK-M7.2-T3                                                      |
| Spec        | SPEC-FR-M7.2                                                       |
| Boundary    | UI Configurator (`ui/configurator/`)                               |
| Status      | VERIFIED                                                           |
| Depends On  | SPEC-FR-M7.2, TASK-M7.2-T1                                         |

## Objective

Implement session-based authentication verification in the React 19 Configurator UI by invoking the BFF `GET /api/v1/auth/me` route at application startup. Implement an authorization guard that protects all administration and editing views, allowing access only to users who possess either the `keeper-admin` or `agent-spawner` roles. Unauthenticated users must be redirected to the authentication flow, and authenticated but unauthorized users must see a styled "Access Denied" page.

## Files

| File | Action |
|------|--------|
| `ui/configurator/src/hooks/useAuth.ts` | NEW |
| `ui/configurator/src/components/AuthGuard.tsx` | NEW |
| `ui/configurator/src/components/AuthGuard.test.tsx` | NEW |
| `ui/configurator/src/App.tsx` | MODIFY |

## RED Phase

1. **Auth Guard Component Tests**:
   - Create `ui/configurator/src/components/AuthGuard.test.tsx` using Vitest and React Testing Library.
   - Mock calls to `GET /api/v1/auth/me` to assert the following behaviors:
     - **Unauthenticated Users**: Mock `/api/v1/auth/me` to return `401 Unauthorized`. Assert that the guard intercepts this and redirects the user by modifying `window.location.href` to the login path (e.g. `/api/v1/auth/login`).
     - **Unauthorized Users**: Mock `/api/v1/auth/me` to return `200 OK` with a user profile that lacks the required roles (e.g. user only has the `keeper-viewer` role). Assert that the application renders a styled "Access Denied" view and blocks the protected children components.
     - **Authorized Users**: Mock `/api/v1/auth/me` to return `200 OK` with either `keeper-admin` or `agent-spawner` inside the roles array. Assert that the children components are rendered correctly without redirection or blocking.
   - Run the frontend tests inside `ui/configurator` using `npm run test` (must fail/error because files do not exist yet).

## GREEN Phase

1. **Implement `useAuth` hook**:
   - Create `ui/configurator/src/hooks/useAuth.ts` to perform a fetch query on startup against `GET /api/v1/auth/me` (configured with `credentials: 'same-origin'` or `include` to ensure cookies are sent).
   - Expose the user profile, authentication state (loading, authenticated, unauthenticated), and list of roles.

2. **Implement `AuthGuard` Component**:
   - Create `ui/configurator/src/components/AuthGuard.tsx` wrapping routes or layouts.
   - If authentication status is loading, render a themed loading indicator/spinner.
   - If unauthenticated, redirect to the BFF authentication entry point (typically `/api/v1/auth/login` or via routing rules).
   - If authenticated but missing both `keeper-admin` and `agent-spawner` roles, show a premium-styled "Access Denied" page with steel/porphyry colors and a logout action.

3. **Integrate into `App.tsx`**:
   - Wrap core administration routes (Wizard, Advanced Settings, list screens) inside the `AuthGuard` wrapper.
   - Provide a logout mechanism (e.g. button in header/navbar) targeting `POST /api/v1/auth/logout` or the OIDC end-session endpoint brokered by the BFF.

4. **Verify tests**:
   - Run Vitest tests (`npm run test`) and ensure the newly added auth tests pass.

## REFACTOR Phase

- Audit `useAuth` caching to avoid multiple concurrent calls to the `/me` endpoint during mounting.
- Ensure that the login/logout target routes are configurable or resolved relative to the host.
- Verify keyboard accessibility on the "Access Denied" page and proper clean-up of redirects during route transition.
