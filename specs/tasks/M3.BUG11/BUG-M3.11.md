# BUG-M3.11: Inconsistent REST API Semantics and Null Empty Collections in List Endpoints

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.11                                                          |
| Status        | CLOSED                                                             |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | internal/keeper/adapters/http/*                                    |
| Violates      | SPEC-NFR-OPENAPI, SPEC-NFR-HTTP                                    |
| Discovered    | API response contract review during OpenAPI validation             |

## Problem Statement

When querying the list endpoints for all primary Keeper aggregates (e.g. `GET /api/v1/llm-bindings`, `GET /api/v1/mcp-servers`, `GET /api/v1/skills`, `GET /api/v1/prompts`, `GET /api/v1/prompt-collections`, `GET /api/v1/agents`, `GET /api/v1/communities`) and there are no records returned for the requesting tenant, the server serializes the response array as a JSON `null` value instead of a clean, empty list `[]`.

This occurs because the underlying slice is initialized as a nil slice (e.g., `var list []Entity`) in Go. When passed directly to the Gin JSON serializer, a nil slice is serialized to the JSON `null` literal. This violates standard REST API guidelines, where list queries should always return a consistent array format (e.g., `[]`).

In addition, other REST semantics need to be audited and verified, including:
1. Returning correct HTTP status codes for empty resources (should be `200 OK` with an empty array `[]` rather than `404 Not Found` or `null`).
2. Correct handling of HTTP status codes for CRUD updates and deletions (e.g. returns `204 No Content` for successful deletes).

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| `http` Handlers | `internal/keeper/adapters/http/` | Slices are returned as nil, causing Gin's JSON serializer to produce `null` rather than `[]` in HTTP responses. |

## Impact

1. **Client-Side Deserialization Failures**: Frontend and API clients that expect a structured array block (and attempt to iterate over it or check its length) crash or throw errors when encountering a `null` value.
2. **OpenAPI Violation**: The live OpenAPI specification states that list endpoints return `array` types. A `null` response violates this API schema contract.
3. **Inconsistent REST Semantics**: Lack of standardized REST response patterns across resource controllers.

## Expected Behaviour

1. List endpoints MUST always return a valid JSON array (`[]`) even when the result set is completely empty.
2. HTTP handlers should explicitly initialize returned slices using `make([]Type, 0)` or ensure that nil slices are mapped to empty slices prior to JSON rendering.
3. REST semantics (status codes, errors, and JSON response shapes) must be fully audited and aligned with architectural patterns.

## Acceptance Criteria

1. Querying any list API endpoint (LLM Bindings, MCP Servers, Skills, Prompts, Prompt Collections, Agents, Communities) when no records exist returns an HTTP `200 OK` with a response payload of `[]`.
2. Response payloads never contain a top-level `null` value for lists.
3. API contract integration tests verify that returned values for empty collections are strictly represented as `[]`.
