# BUG-M3.14: POST REST Calls Missing Location HTTP Header

| Field         | Value                                                              |
|---------------|--------------------------------------------------------------------|
| ID            | BUG-M3.14                                                          |
| Status        | OPEN                                                               |
| Severity      | MEDIUM                                                             |
| Milestone     | M3 — Keeper Core                                                   |
| Affects       | `internal/keeper/adapters/inbound/http/*`                         |
| Violates      | SPEC-NFR-HTTP                                                      |
| Discovered    | Manual API design audit                                            |

## Problem Statement

When creating resources via `POST` REST operations (such as Agents, Communities, LLM Bindings, MCP Servers, Prompts, Skills, and Skill Collections), the server responds with `201 Created` and a `null` body. However, it does not set the standard REST `Location` HTTP header pointing to the newly created resource (e.g. `/api/v1/agents/{id}`). The client has no standardized way of resolving the newly created resource's UUID/URI without parsing custom metadata or initiating list operations, which violates strict REST conventions.

## Affected Components and Files

| Component / File | Location | Issue |
|------------------|----------|-------|
| HTTP Handlers | `internal/keeper/adapters/inbound/http/*` | Successful `POST` handlers do not call `c.Header("Location", ...)` before returning `201 Created`. |

## Impact

1. **Non-standard REST API**: Clients cannot fetch resource details directly using the created location.
2. **Contract Violation**: Violates API-first development specifications where state mutations must yield direct pointers to new resources.

## Expected Behaviour

1. Every successful resource-creating `POST` endpoint MUST include a `Location` HTTP header containing the URI of the newly created resource (e.g. `/api/v1/agents/<uuid>`).
2. The HTTP response status must remain `201 Created` with a `null` body.

## Acceptance Criteria

1. Unit tests verify that creating an Agent, Community, LLM Binding, MCP Server, Prompt, Skill, or Skill Collection via `POST` returns a `Location` header matching `/api/v1/<resource>/<uuid>`.
2. OpenAPI specs are in parity with the updated HTTP response headers.
