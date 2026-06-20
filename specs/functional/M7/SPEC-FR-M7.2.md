# SPEC-FR-M7.2: Configurator UI

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.2                                |
| Status        | ACCEPTED                                    |
| Milestone     | M7                                          |
| Component     | ui                                          |
| Depends On    | SPEC-FR-M7.1, SPEC-FR-M7.4                  |
| Supersedes    | none                                        |

## Context

Web interface for administrators to manage agents, communities, prompts, and skills. It is a separate React 19 application decoupled from the Auditor UI.

## Specification

1. The Configurator UI MUST be built with React 19 (per SPEC-NFR-STACK).
2. The UI MUST be a separate deployment (not bundled with the Auditor UI).
3. The UI MUST support session-based authentication brokered by the BFF. It MUST invoke `GET /api/bff/v1/auth/me` on startup to verify authentication and fetch user capabilities (roles: `keeper-admin`, `agent-spawner`).
4. The UI MUST provide CRUD interfaces for: agents, communities, and agent-community assignments, calling the BFF route namespace `/api/bff/v1/configurator/`. Creation and modification workflows MUST be guided by a step-by-step **Wizard-driven interface** for common configurations, but also provide an **Advanced Settings panel** that exposes low-level raw schema/JSON editing.
5. The UI MUST provide a community topology visualization component supporting multiple layouts: **standalone** units, **hub-spoke** networks, and extensible formats representing **serialized** future topologies.

## Acceptance Criteria

1. **Static Isolation**: Served as a separate single-page application.
2. **Access Control**: Users without the `keeper-admin` or `agent-spawner` roles (as returned by the `/me` endpoint) are blocked from accessing editing views.
3. **BFF Routing**: All API calls from the UI target the BFF namespace `/api/bff/v1/configurator/` with the browser automatically attaching the secure session cookie.

## Test Plan

### Frontend Unit & Component Tests
* **Auth Guard Component**: Test that unauthorized users are redirected to the login endpoint, and authorized users can access the dashboard.
* **CRUD Screens**: Test form validations, submission states, and error handling for agent/community creation.
* **Topology Visualization**: Verify correct rendering of nodes and links based on a mock graph payload.

## Files Affected

* `ui/configurator/src/` (React 19 application components, hooks, and routing)
* `ui/configurator/package.json` (React 19 configurations)
* `deploy/helm/tacito-square/templates/ui-configurator/` (Deployment and service chart definitions)

