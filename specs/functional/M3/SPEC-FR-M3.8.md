# SPEC-FR-M3.8: PostgreSQL Persistence & Migrations

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M3.8                                |
| Status        | DRAFT                                       |
| Milestone     | M3                                          |
| Component     | keeper                                      |
| Depends On    | SPEC-FR-M3.1, SPEC-FR-M3.2, SPEC-FR-M3.3, SPEC-FR-M3.4, SPEC-FR-M3.5, SPEC-FR-M3.6, SPEC-FR-M3.7 |
| Supersedes    | none                                        |

## Context

Agent and community definitions need durable persistence in PostgreSQL. This spec covers the repository adapter implementing outbound ports, and the database migration strategy.

## Specification

1. The system MUST use `pgx` as the PostgreSQL driver (per SPEC-NFR-STACK).
2. The system MUST use `goose` for database migrations (per SPEC-NFR-STACK).
3. Repository adapters MUST implement outbound ports defined in the domain layer (per SPEC-NFR-HEXAGONAL).
4. Migrations MUST be idempotent and versioned.
5. Connection pooling MUST be configured with sensible defaults and exposed as configuration.

## Acceptance Criteria

To be defined during spec review.

## Test Plan

To be defined during spec review.

## Files Affected

To be defined during spec review.
