# Milestone M7: BFF & UIs

| Field      | Value |
|------------|-------|
| Status     | ⏸️ SUSPENDED |

## Goal

Admin (Configurator) and monitoring (Auditor) web interfaces with OIDC authentication, bridged to keeper APIs through a BFF layer.

## Deliverable

Login via Keycloak → browse communities → spawn agent → interact with agent → view audit log. Two separate React 19 deployments.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M7.1 | BFF API Bridge Layer | bff | SPEC-FR-M2.6 |
| SPEC-FR-M7.2 | Configurator UI | ui | SPEC-FR-M7.1 |
| SPEC-FR-M7.3 | Auditor UI | ui | SPEC-FR-M7.1 |
| SPEC-FR-M7.4 | OIDC Login Flow (Keycloak) | bff | SPEC-FR-M7.1, SPEC-FR-M3.9 |
