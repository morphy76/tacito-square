# SPEC-NFR-OPENAPI: Live OpenAPI Endpoints

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-OPENAPI                   |
| Status        | ACCEPTED                           |
| Component     | keeper, agent, bff                 |

## Specification

1. Every artifact that exposes an HTTP API MUST serve its OpenAPI 3.x spec at `GET /openapi.json`.
2. The spec MUST be generated from code annotations or maintained as a static file embedded at build time.
3. A Swagger UI MAY be served at `GET /swagger/` in dev mode (disabled in production via config).
4. The served spec MUST match the contract in `api/openapi/`.
5. Contract tests (FR-12.5) MUST validate that the live endpoint matches the committed spec file.

### Domain and Boundary Tagging

6. Every OpenAPI spec MUST declare a top-level `tags` array that maps the API surface to its domain model.
7. Tags MUST follow the naming convention `domain/subdomain`, where:
   - `domain` is the bounded context (e.g. `infrastructure`, `agent-config`, `community`)
   - `subdomain` is the capability cluster within that context (e.g. `llm-bindings`, `skills`, `prompt-engineering`)
8. Every operation MUST carry exactly one tag from the declared `tags` array.
9. Each tag entry MUST include a `description` that names the bounded context and its responsibility.
10. Tag names MUST be stable across versions; adding a new tag is non-breaking, renaming one is a breaking change.

## Acceptance Criteria

1. `GET /openapi.json` returns valid OpenAPI 3.x JSON on keeper, agent, bff
2. Swagger UI accessible in dev mode
3. CI fails if live spec diverges from committed contract
4. Every operation in the spec carries exactly one `domain/subdomain` tag
5. All declared tags appear in the top-level `tags` array with a non-empty `description`
