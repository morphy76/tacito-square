# Milestone M8: UIs & BFF

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Admin and end-user web interfaces with OIDC authentication.

## Deliverable

Login → browse communities → spawn agent → chat with agent → view audit log.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-08.6 | BFF scaffolding (Gin) | FR-08.6 | bff | — |
| SPEC-FR-08.1-v2 | Keeper BFF routes | FR-08.1 | bff | SPEC-FR-08.6 |
| SPEC-FR-08.2 | User BFF routes | FR-08.2 | bff | SPEC-FR-08.6 |
| SPEC-FR-08.3 | Keeper UI (React 19) | FR-08.3 | ui | SPEC-FR-08.1-v2 |
| SPEC-FR-08.4 | User UI (React 19) | FR-08.4 | ui | SPEC-FR-08.2 |
| SPEC-FR-08.5-v2 | OIDC login flow (Keycloak integration) | FR-08.5 | bff | SPEC-FR-13.1 |
| SPEC-FR-13.7 | Service-to-service auth | FR-13.7 | keeper | SPEC-FR-03.2 |
