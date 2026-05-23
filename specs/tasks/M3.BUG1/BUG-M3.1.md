# BUG-M3.1: Supporting Entity Models Lack Tenant Segregation

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.1                                                           |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | TASK-M3.1.1, TASK-M3.2.1, TASK-M3.3.1, TASK-M3.4.1              |
| Violates      | SPEC-NFR-MULTITENANCY §2 (Shared Multi-Tenant Units)              |
| Discovered    | Post-implementation review against SPEC-NFR-MULTITENANCY          |

## Problem Statement

The domain models for all four supporting entities implemented in M3 — **LLM Provider Bindings** (M3.1), **MCP Servers** (M3.2), **Skill Collections** (M3.3), and **Prompt Collections / Templates** (M3.4) — do not carry a `TenantID` field.

Keeper is classified as a **Shared Multi-Tenant Unit** under SPEC-NFR-MULTITENANCY, which means it serves multiple tenants from a single running process and must dynamically resolve and enforce tenant identity on every request. Without a `tenant_id` on each aggregate root, the persistence layer cannot scope reads and writes to the correct tenant, resulting in full cross-tenant data visibility across all API operations.

## Affected Aggregates and Files

| Aggregate          | Domain file                                         | Missing field  |
|--------------------|-----------------------------------------------------|----------------|
| `LLMBinding`       | `internal/keeper/domain/llm_binding.go`             | `TenantID`     |
| `MCPServer`        | `internal/keeper/domain/mcp_server.go`              | `TenantID`     |
| `Skill`            | `internal/keeper/domain/skill.go`                   | `TenantID`     |
| `PromptTemplate`   | `internal/keeper/domain/prompt.go`                  | `TenantID`     |
| `PromptCollection` | `internal/keeper/domain/prompt.go`                  | `TenantID`     |

## Impact

1. **Data isolation broken**: any authenticated (or unauthenticated) request to `GET /api/v1/llm-bindings`, `/mcp-servers`, `/skills`, `/prompts`, or `/prompt-collections` returns records belonging to **all** tenants, with no filtering boundary.
2. **Write operations unscoped**: `POST` and `PUT` handlers persist new entities without stamping a tenant owner, making future retrospective scoping impossible without a destructive migration.
3. **Compliance failure**: the system cannot satisfy SPEC-NFR-MULTITENANCY AC-2 ("if tenant resolution fails or is missing, requests must be rejected") because the domain layer provides no anchor for tenant-aware enforcement.

## Root Cause

The M3.1–M3.4 implementation tasks (TASK-M3.1.1, TASK-M3.2.1, TASK-M3.3.1, TASK-M3.4.1) specified domain model and repository boundaries without cross-referencing SPEC-NFR-MULTITENANCY. The NFR was accepted before M3 started but was not propagated as a constraint into the per-entity task acceptance criteria, leaving `tenant_id` out of the data model and all downstream artefacts (Postgres migrations, repository queries, HTTP handlers).

## Expected Behaviour (per SPEC-NFR-MULTITENANCY)

1. Every aggregate root persisted by Keeper MUST carry a non-nullable `TenantID uuid.UUID` field that is stamped at creation time from the resolved tenant context.
2. All repository list and get queries MUST include a `WHERE tenant_id = $1` predicate derived from `context.Context`.
3. All repository create and update operations MUST set `tenant_id` from the tenant context, never from the request payload.
4. HTTP handlers MUST propagate the tenant resolved by the authentication middleware into `context.Context` before calling any application or repository port.
5. Postgres schema migrations for all five tables MUST add a non-nullable `tenant_id UUID NOT NULL` column with an index.

## Acceptance Criteria

1. `LLMBinding`, `MCPServer`, `Skill`, `PromptTemplate`, and `PromptCollection` each declare a `TenantID uuid.UUID` field.
2. Repository adapters for all five entities filter every read query by `tenant_id` extracted from the Go `context.Context`.
3. Repository create adapters for all five entities set `tenant_id` from the Go `context.Context`; the field is never accepted from HTTP request payloads.
4. New Postgres migration scripts add `tenant_id UUID NOT NULL` plus a covering index to each affected table.
5. Domain unit tests and repository integration tests assert tenant scoping: a record created under tenant A is not visible to tenant B.
6. No existing passing tests are broken by the fix.
