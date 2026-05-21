# SPEC-FR-M7.3: Auditor UI

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.3                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | ui                                          |
| Depends On    | SPEC-FR-M7.1                                |
| Supersedes    | none                                        |

## Context

Web interface for monitoring agent activity, conversation logs, audit trails, and system health. Separate React 19 deployment focused on observability.

## Specification

1. The Auditor UI MUST be built with React 19 (per SPEC-NFR-STACK).
2. The UI MUST be a separate deployment (not bundled with the Configurator UI).
3. The UI MUST support OIDC login via Keycloak (roles: keeper-admin, keeper-viewer).
4. The UI MUST display real-time agent status per community.
5. The UI MUST provide a conversation log viewer with thread navigation.
6. The UI MUST provide audit trail search and filtering.
7. The UI SHOULD display community health and quota usage dashboards.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
