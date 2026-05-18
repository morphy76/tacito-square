# Milestone M3: Deployable Core

| Field      | Value |
|------------|-------|
| Status     | ⬜ PLANNED |

## Goal

Make the existing M1/M2 codebase deployable to production with real persistence, secured APIs, and production-grade Helm values.

## Deliverable

`helm install` on a real cluster → secure, persistent, observable single-agent system with RBAC-protected APIs.

## Specs Required

| Spec ID | Title | FR Ref | Component | Depends On |
|---------|-------|--------|-----------|------------|
| SPEC-FR-03.1 | PostgreSQL AgentStore (pgx, migrations) | FR-01.1 | keeper | SPEC-FR-01.1 |
| SPEC-FR-13.1 | RBAC role model enforcement | FR-13.1 | keeper, shared | SPEC-FR-08.5 |
| SPEC-FR-03.2 | Gin RBAC middleware (JWT → Keycloak JWKS) | FR-03.2 | shared | SPEC-FR-13.1 |
| SPEC-FR-13.3 | Principal logging (trace_id + subject + roles) | FR-13.3 | shared | SPEC-NFR-LOG |
| SPEC-FR-13.6 | Role-based route protection | FR-13.6 | keeper | SPEC-FR-03.2 |
| SPEC-FR-M3-PROD | Production Helm values (TLS, secrets, HA) | — | deploy | SPEC-FR-12.6 |
