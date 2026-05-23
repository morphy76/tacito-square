# BUG-M3.1: Supporting Entity Models Lack Tenant Segregation

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.1                                                           |
| Status        | CLOSED                                                             |
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

1. **Data isolation broken**: any request to `GET /api/v1/llm-bindings`, `/mcp-servers`, `/skills`, `/prompts`, or `/prompt-collections` returned records belonging to **all** tenants.
2. **Write operations unscoped**: `POST` and `PUT` handlers persisted new entities without stamping a tenant owner.
3. **Compliance failure**: the system could not satisfy SPEC-NFR-MULTITENANCY AC-2.

## Expected Behaviour (per SPEC-NFR-MULTITENANCY & User Feedback)

1. Every aggregate root persisted by Keeper MUST carry a non-nullable `TenantID string` field, representing the canonical `FullName()` format (`tenantId-subscriptionId`), which is stamped at creation/update time from the resolved tenant context.
2. All repository list, get, update, and delete queries MUST be strictly filtered by `tenant_id` derived from `context.Context`.
3. HTTP handlers MUST validate and propagate the tenant resolved by the extensible `TenantResolutionMiddleware` into `context.Context`.
4. Postgres schema migrations (edited directly since the database is not yet provisioned in production) MUST add non-nullable `tenant_id VARCHAR(255) NOT NULL` columns, indexes, and composite uniqueness constraints qualified by `(tenant_id, name)`.

## Acceptance Criteria

1. `LLMBinding`, `MCPServer`, `Skill`, `PromptTemplate`, and `PromptCollection` each declare a `TenantID string` field.
2. Repository adapters for all five entities filter every read query by `tenant_id` extracted from the Go `context.Context`.
3. Repository create/update adapters set `tenant_id` from the Go `context.Context`; the field is never accepted from HTTP request payloads.
4. Postgres migrations add `tenant_id VARCHAR(255) NOT NULL` plus composite unique bounds to each affected table.
5. Domain unit tests and repository integration tests assert tenant scoping: a record created under tenant A is not visible to tenant B.
6. No existing passing tests are broken by the fix.

## Resolution Details

The bug is fully resolved via TDD (Red-Green-Refactor) approach:
1. **Domain Models**: Added `TenantID string` to all five aggregates, with strict invariant validation checks in `Validate()` rejecting empty tenant strings.
2. **HTTP Middleware**: Implemented an extensible policy pattern using a `TenantResolver` interface, equipped with a standard `HeaderTenantResolver` policy reading `X-Tenant-ID` and `X-Subscription-ID` headers.
3. **Controller & Handler Updates**: Stamped HTTP request payloads with resolved tenant credentials before checking invariants, preserving clean decoupling.
4. **Direct DB Migrations**: Modified migrations `00001` through `00004` directly, introducing `tenant_id VARCHAR(255) NOT NULL` columns, indexes, and composite uniqueness bounds (unique per-tenant names).
5. **Context-Scoped Repository Adapters**: Updated all database repositories to fetch tenant IDs from context and securely segregate reads, writes, joins, and relationships.
6. **Multi-Tenant Integration Tests**: Added robust subtests asserting comprehensive multi-tenant isolation, data visibility, and composite uniqueness boundaries across all five aggregates.

Closed: 2026-05-23
Approver: USER (approved implementation_plan.md)

