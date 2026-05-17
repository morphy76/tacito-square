# SPEC-FR-12.2: API-First Design

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-12.2                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-12.2                            |
| Component     | all                                |
| Depends On    | —                                  |

## Context

UIs are optional consumers. All functionality MUST be accessible via authenticated REST APIs. This ensures CLI, SDK, and automated pipeline access without UI dependency.

## Specification

1. Every feature MUST be exposed via a REST API endpoint before any UI is built.
2. APIs MUST use versioned paths: `/api/v1/...`.
3. Response format MUST be JSON.
4. Error format MUST be `{"error": "message"}`.
5. Success responses MUST use appropriate HTTP status codes (200, 201, 204).
6. UIs (M6) MUST consume the same APIs — no UI-only backend routes.

## Acceptance Criteria

1. Keeper agents are manageable entirely via curl/REST
2. All endpoints use `/api/v1/` prefix
3. JSON response format consistent
4. No functionality locked behind UI

## Files

- `internal/keeper/adapters/inbound/httphandler/handler.go` ✅ (routes: GET/POST/DELETE under /api/v1/)
