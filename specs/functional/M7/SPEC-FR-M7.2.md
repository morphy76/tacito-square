# SPEC-FR-M7.2: Configurator UI

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M7.2                                |
| Status        | DRAFT                                       |
| Milestone     | M7                                          |
| Component     | ui                                          |
| Depends On    | SPEC-FR-M7.1                                |
| Supersedes    | none                                        |

## Context

Web interface for administrators to manage agents, communities, prompts, and skills. Separate React 19 deployment from the Auditor UI.

## Specification

1. The Configurator UI MUST be built with React 19 (per SPEC-NFR-STACK).
2. The UI MUST be a separate deployment (not bundled with the Auditor UI).
3. The UI MUST support OIDC login via Keycloak (roles: keeper-admin, agent-spawner).
4. The UI MUST provide CRUD screens for: agents, communities, agent-community assignments.
5. The UI SHOULD provide community topology visualization (hub-spoke graph).

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
