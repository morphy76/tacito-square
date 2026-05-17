# SPEC-NFR-OPENAPI: Live OpenAPI Endpoints

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-OPENAPI                   |
| Status        | DRAFT                              |
| Milestone     | M9                                 |
| FR/NFR Ref    | FR-12.4                            |
| Component     | keeper, agent, bff                 |

## Specification

1. Every artifact that exposes an HTTP API MUST serve its OpenAPI 3.x spec at `GET /openapi.json`.
2. The spec MUST be generated from code annotations or maintained as a static file embedded at build time.
3. A Swagger UI MAY be served at `GET /swagger/` in dev mode (disabled in production via config).
4. The served spec MUST match the contract in `api/openapi/`.
5. Contract tests (FR-12.5) MUST validate that the live endpoint matches the committed spec file.

## Acceptance Criteria

1. `GET /openapi.json` returns valid OpenAPI 3.x JSON on keeper, agent, bff
2. Swagger UI accessible in dev mode
3. CI fails if live spec diverges from committed contract
