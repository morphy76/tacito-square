# Milestone M3: Keeper Core

| Field      | Value |
|------------|-------|
| Status     | ✔️ IMPLEMENTED |

## Goal

Keeper manages agents and communities via authenticated REST API, persists state to PostgreSQL, and submits Agent CRDs to the K8s API for operator reconciliation.

## Deliverable

Authenticated API calls → create community → create agent → assign agent to community → TacitoAgent CRD submitted to K8s API.

## Specs Required

| Spec ID | Title | Component | Depends On |
|---------|-------|-----------|------------|
| SPEC-FR-M3.1 | LLM Provider Bindings & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.2 | MCP Servers & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.3 | Skill Collections & CRUD API | keeper | SPEC-FR-M2.3, SPEC-FR-M3.2 |
| SPEC-FR-M3.4 | Prompt Collections & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.5 | Agent Domain Model & CRUD API | keeper | SPEC-FR-M2.3, SPEC-FR-M3.1, SPEC-FR-M3.2, SPEC-FR-M3.3, SPEC-FR-M3.4 |
| SPEC-FR-M3.6 | Community Domain Model & CRUD API | keeper | SPEC-FR-M2.3 |
| SPEC-FR-M3.7 | Agent-Community Assignment | keeper | SPEC-FR-M3.5, SPEC-FR-M3.6 |
| SPEC-FR-M3.8 | PostgreSQL Persistence & Migrations | keeper | SPEC-FR-M3.1, SPEC-FR-M3.2, SPEC-FR-M3.3, SPEC-FR-M3.4, SPEC-FR-M3.5, SPEC-FR-M3.6, SPEC-FR-M3.7 |
