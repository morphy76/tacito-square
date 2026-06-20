# SPEC-FR-M7.3: Auditor UI

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.3                                |
| Status        | ACCEPTED                                    |
| Milestone     | M7                                          |
| Component     | ui                                          |
| Depends On    | SPEC-FR-M7.1, SPEC-FR-M7.4                  |
| Supersedes    | none                                        |

## Context

Web interface for monitoring agent activity, conversation logs, audit trails, and system health. It is a separate React 19 application focused on observability.

## Specification

1. The Auditor UI MUST be built with React 19 (per SPEC-NFR-STACK).
2. The UI MUST be a separate deployment (not bundled with the Configurator UI).
3. The UI MUST support session-based authentication brokered by the BFF. It MUST invoke `GET /api/bff/v1/auth/me` on startup to verify credentials and roles (roles: `keeper-admin`, `keeper-viewer`).
4. The UI MUST display real-time agent status per community, listening to the BFF SSE gateway `/api/bff/v1/events/stream`.
5. The UI MUST provide a conversation log viewer with thread navigation, querying the BFF namespace `/api/bff/v1/auditor/`.
6. The UI MUST provide audit trail search and filtering.
7. The UI SHOULD display community health and quota usage dashboards.

## Acceptance Criteria

1. **Static Isolation**: Served as a separate single-page application.
2. **Access Control**: Block users without `keeper-admin` or `keeper-viewer` roles from reading system audit details.
3. **SSE Connection Persistence**: Automatically connects to the BFF SSE gateway to update active thread displays and agent status indicators in real time without refreshing.
4. **BFF Routing**: All API calls from the UI target the BFF namespace `/api/bff/v1/auditor/`.

## Test Plan

### Frontend Unit & Component Tests
* **Auth Guard Component**: Verify routing restriction based on `/me` roles.
* **SSE Event Receiver**: Mock the Server-Sent Events stream from the BFF and verify the UI updates the agent status list in real time.
* **Audit Trail Filter**: Verify that filters (tenant, agent, timestamp range) generate correct URL queries and display search results properly.

## Files Affected

* `ui/auditor/src/` (React 19 application components, observability charts, hooks, and routing)
* `ui/auditor/package.json` (React 19 configurations)
* `deploy/helm/tacito-square/templates/ui-auditor/` (Deployment and service chart definitions)

