# SPEC-FR-M9.8: Compact JSON API Responses

| Field         | Value                                       |
|---------------|---------------------------------------------|
| ID            | SPEC-FR-M9.8                                |
| Status        | ACCEPTED                                    |
| Milestone     | M9                                          |
| Component     | keeper, shared                              |
| Depends On    | none                                        |
| Supersedes    | none                                        |

## Context

In REST API responses, returning `null` attributes, empty objects, or empty arrays is verbose and consumes unnecessary network bandwidth. 
To provide compact JSON responses, any optional or empty slices, maps, pointers, and interfaces should be omitted from the serialized JSON output.

## Specification

1. All REST API JSON response payloads served by the Keeper component MUST omit:
   * Attributes with `null` values (such as unused optional fields or empty pointers).
   * Empty arrays/slices (such as empty list of skills or MCP clients).
   * Empty objects/maps (such as empty custom environment maps).
2. This MUST be achieved by appending the `,omitempty` option to the corresponding `json:"..."` struct tags in Go.
3. This change applies to the primary Keeper domain model structs and package-exposed API contract structs.

## Acceptance Criteria

1. Serializing a model or response with empty/nil slices (e.g. `skills`, `mcp_clients`, `templates`, `agents`), empty/nil maps (e.g. `env`, `configuration`), or nil pointers (e.g. `community_id`) omits those fields from the output JSON.
2. Response validation and contract compatibility tests pass successfully.
3. Empty collections at the top level of list endpoints (e.g. `GET /api/v1/agents`) MUST still return a clean empty array `[]` as defined by `BUG-M3.11`. Only fields *within* the response objects should be omitted if they are empty/null.

## Test Plan

### Automated Tests
1. **Contract/Unit Tests:**
   * Run the test suite:
     ```bash
     make test
     ```
   * Run contract tests to ensure no schema drift or field matching failures:
     ```bash
     make test-contract
     ```
2. **Serialization Tests:**
   * Verify that marshalling a model with empty slices (e.g. nil or empty skills) does not produce the key in the JSON output.

## Files Affected

* `[MODIFY] internal/keeper/domain/model/agent.go`
* `[MODIFY] internal/keeper/domain/model/community.go`
* `[MODIFY] internal/keeper/domain/model/community_card.go`
* `[MODIFY] internal/keeper/domain/model/mcp_client.go`
* `[MODIFY] internal/keeper/domain/model/prompt.go`
* `[MODIFY] internal/keeper/domain/model/skill.go`
* `[MODIFY] internal/keeper/domain/model/echo.go`
* `[MODIFY] pkg/agentcard/agent_card.go`
