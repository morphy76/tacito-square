# Milestone M3: Keeper Core

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Keeper manages agents and communities via authenticated REST API, persists state to PostgreSQL, and submits Agent CRDs to the K8s API for operator reconciliation.

## Deliverable

Authenticated API calls → create community → create agent → assign agent to community → TacitoAgent CRD submitted to K8s API.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M3.1 | Agent Domain Model & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.2 | Community Domain Model & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.3 | Agent-Community Assignment | keeper | SPEC-FR-M3.1, SPEC-FR-M3.2 |
| SPEC-FR-M3.4 | PostgreSQL Persistence & Migrations | keeper | SPEC-FR-M3.1, SPEC-FR-M3.2 |
| SPEC-FR-M3.5 | OIDC/JWT Authentication | keeper, shared | SPEC-FR-M2.2 |
| SPEC-FR-M3.6 | Agent CRD Submission | keeper | SPEC-FR-M3.3, SPEC-FR-M4.1 |
