# BUG-M3.15: POST REST Calls Lack Cancel Context

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.15                                                          |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | `internal/keeper/adapters/inbound/http/*`                         |
| Violates      | SPEC-NFR-REACTIVE, SPEC-NFR-CLOUD                                  |
| Discovered    | Code review during load testing                                     |

## Problem Statement

When handling `POST` REST operations in our inbound HTTP adapters, the handlers extract the parent request context `c.Request.Context()` but do not wrap it in an explicit cancelable context (e.g. `context.WithCancel` or `context.WithTimeout`). If an HTTP client disconnects abruptly mid-request, or if a downstream datastore operation hangs, the execution context is not cancelled dynamically. This prevents early resource reclamation, holding up postgres transaction sessions, Redis cache connections, and trace spans, which violates strict reactive resource cleanup principles.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| HTTP Handlers | `internal/keeper/adapters/inbound/http/*` | Inbound `POST` handlers propagate `c.Request.Context()` directly without creating a cancelable context via `context.WithCancel(ctx)` and deferring `cancel()`. |

## Impact

1. **Resource Leaks**: Hanging database queries or client disconnections leave transactions active until they hit standard database timeouts.
2. **Goroutine Accumulation**: Goroutines handling these mutations do not exit promptly on client cancellation, potentially causing thread accumulation under load.

## Expected Behaviour

1. All resource-creating `POST` endpoint handlers MUST create a cancelable context using `context.WithCancel` derived from `c.Request.Context()`.
2. The handlers MUST defer the context cancellation function (`cancel()`) immediately to ensure resources are returned to connection pools and OTel tracer spans close gracefully when the request completes or is cancelled.

## Acceptance Criteria

1. Inbound HTTP `POST` handler implementations for all resources contain `context.WithCancel` wrappers.
2. Unit tests verify that handler functions successfully execute and clean up their execution contexts upon completion or client disconnect.
