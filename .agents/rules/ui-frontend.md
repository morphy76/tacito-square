---
trigger: glob
globs: ["ui/**/*.{ts,tsx,js,jsx,css,json}"]
description: Frontend engineering guidelines for React 19, TypeScript strictness, component boundaries, and Vitest testing in ui/.
---

# UI & Frontend Engineering Guidelines (React 19 & TypeScript)

This rule governs all code changes within the `ui/` directory (specifically `ui/configurator/`), enforcing React 19 standards, TypeScript strictness, component isolation, and automated testing.

## 1. React 19 & Modern Idioms

- **Functional Components & Hooks**: All components MUST be written as functional components using modern React hooks. Never use legacy class components.
- **React Compiler Compatibility**:
  - Avoid unnecessary `useMemo` and `useCallback` when expressions are lightweight; let the React 19 Compiler optimize re-renders.
  - Never mutate state or props directly. Maintain immutable state updates.
- **Hook Hygiene**:
  - Avoid using `useEffect` for state synchronization or derived state. Compute derived values directly in the render phase.
  - Reserve `useEffect` strictly for external system synchronization (e.g. event listeners, SSE subscriptions, browser APIs).
- **Form Handling**: Use uncontrolled inputs with `FormData` / React 19 form actions or controlled state with minimal re-render scope.

## 2. TypeScript Strictness

- **Explicit Interfaces & Contracts**: Declare clear TypeScript interfaces for all component props, API request/response payloads, and state models under `src/types/` or co-located with components.
- **Zero `any` Policy**: Never use `any`. Use strict union types, generics, or `unknown` with type guards.
- **Strict Null Checks**: Handle optional or nullable values explicitly with optional chaining (`?.`) and nullish coalescing (`??`).

## 3. Component Architecture & Directory Structure

Organize the `ui/` codebase by logical feature and presentation boundaries:

```text
ui/configurator/src/
├── components/          # Reusable, presentational UI components (Forms, Tables, Visualizers)
│   ├── AgentCRUD/       # Feature-specific components and sub-wizards
│   ├── Topology/        # Dynamic topology & community visualizers
│   └── common/          # Shared atomic widgets (Buttons, Badges, Modals)
├── hooks/               # Custom reusable React hooks
├── services/            # API client layer (REST, GraphQL, SSE)
├── types/               # TypeScript domain type definitions
└── test/                # Test utilities and mock fixtures
```

- **Separation of Concerns**: Separate visual presentation from data fetching. Components should receive data via props or custom hooks, not embed raw fetch calls directly in complex JSX hierarchies.

## 4. Testing Guidelines (Vitest + Testing Library)

- **Test Co-location**: Place component unit tests directly adjacent to the component file (`<Component>.test.tsx`).
- **User-Centric Testing**: Test components from the user's perspective using `@testing-library/react` and `@testing-library/user-event`:
  - Query by role, label text, or placeholder (`screen.getByRole`, `screen.getByLabelText`).
  - Avoid querying internal component state or implementation details.
- **Mocking & Isolation**: Mock API services and network calls cleanly using Vitest mocks (`vi.fn()`, `vi.mock()`).
- **Test Command**: All UI tests must pass cleanly:
  ```bash
  cd ui/configurator && npm run test
  ```

---

## Developer Checklists & Verifications

- [ ] Are all new components written as functional components with typed props?
- [ ] Is `any` completely absent from all TypeScript files?
- [ ] Are unit tests provided in `*.test.tsx` files for new components?
- [ ] Does `npm run test` and `npm run lint` pass in `ui/configurator`?
