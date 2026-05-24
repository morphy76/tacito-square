# BUG-M3.10: Inconsistent Logging of Trace ID and Tenant Context Across Keeper Entities

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.10                                                          |
| Status        | OPEN                                                               |
| Severity      | HIGH                                                               |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/adapters/http/*, internal/shared/observability/*  |
| Violates      | SPEC-NFR-OBSERVABILITY §2 (Trace Correlation), SPEC-NFR-MULTITENANCY §3 |
| Discovered    | Log review during multi-tenant integration testing                 |

## Problem Statement

Logging across different Keeper entities is inconsistent. When handling API calls for various aggregates (LLM Bindings, MCP Servers, Skills, Prompts, Agents, Communities), the structured `zerolog` log messages written to `stdout` do not consistently include crucial correlation context:
1. **Trace ID**: Some logs fail to print the trace ID even when the request is executed within a traced context.
2. **Tenant ID / context**: Several domain-level and adapter-level log statements print messages without stamping the resolved tenant context (such as the tenant's `FullName()`), making it extremely difficult to isolate and trace events belonging to a specific tenant in production.

This hampers observability and log segregation under SPEC-NFR-OBSERVABILITY and SPEC-NFR-MULTITENANCY.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| `http` Handlers | `internal/keeper/adapters/http/` | Entity HTTP controllers do not consistently propagate trace and tenant context fields into logger references. |
| `shared` Observability | `internal/shared/observability/` | Helper functions do not enforce the automatic binding of trace and tenant context in structured logs. |

## Impact

1. **Broken Trace Correlation**: Distributed traces cannot be fully linked with application logs, creating gaps in system-wide debugging.
2. **Multi-Tenant Debugging Gaps**: Operations on entities (e.g. creations, updates, or errors) are logged anonymously without tenant context, making it impossible for operators to filter logs per-tenant.
3. **Auditability Failure**: Inability to verify tenant action segregation through log parsing.

## Expected Behaviour

1. All structured log messages generated inside Keeper HTTP handlers, database adapters, and domain validators MUST consistently carry `trace_id` and `tenant_id` fields if present in the request context.
2. The logging middleware and logger context builders MUST automatically extract trace IDs and tenant contexts from `context.Context` and bind them to the active logger instance.

## Acceptance Criteria

1. Every entity CRUD operation log (LLM Bindings, MCP Servers, Skills, Prompts, Agents, Communities) prints a `trace_id` field.
2. Every entity CRUD operation log prints a `tenant_id` field.
3. Unit and integration tests assert trace and tenant propagation in context-scoped loggers.
