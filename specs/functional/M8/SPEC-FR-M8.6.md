# SPEC-FR-M8.6: Audit Trail (events + queries)

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M8.6                                |
| Status        | DRAFT                                       |
| Milestone     | M8                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.8                                |
| Supersedes    | none                                        |

## Context

All significant events must be recorded for accountability, compliance, and debugging.

## Specification

1. The system MUST define an `AuditEvent` entity: ID, timestamp, actor, action, target, details (JSON), trace ID.
2. Audit events MUST be stored in an append-only PostgreSQL table.
3. Events MUST be emitted for: agent CRUD, community CRUD, assignments, CRD submissions, HITL, login/logout.
4. Query API: `GET /api/v1/audit?actor=&action=&target=&from=&to=` with pagination.
5. Audit events MUST NOT be deletable via API (append-only).
6. Records MUST include OpenTelemetry trace ID for correlation.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
