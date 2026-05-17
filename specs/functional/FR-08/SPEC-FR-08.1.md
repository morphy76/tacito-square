# SPEC-FR-08.1: Keeper REST API

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-FR-08.1                       |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| FR/NFR Ref    | FR-08.1                            |
| Component     | keeper                             |
| Depends On    | SPEC-FR-01.1, SPEC-NFR-HTTP       |

## Context

The Keeper control plane is API-first. All agent lifecycle operations are exposed as RESTful endpoints.

## Specification

1. `RegisterRoutes(*gin.Engine)` MUST register all routes centrally.
2. Endpoints:
   - `GET /healthz` → 200 `{"status": "ok"}`
   - `GET /readyz` → 200 `{"status": "ready"}`
   - `GET /api/v1/agents` → 200 with JSON array of `AgentResponse`
   - `POST /api/v1/agents` → 201 with `AgentResponse` (Gin binding validation)
   - `DELETE /api/v1/agents/:id` → 204 No Content
3. Request validation MUST use `ShouldBindJSON` with `binding:"required"` tags.
4. Errors MUST return `gin.H{"error": message}` format.
5. Path params MUST use `c.Param("id")`.

## API Contract

### POST /api/v1/agents

Request:
```json
{"name": "...", "community_id": "...", "prompt_id": "...", "skill_ids": ["..."]}
```

Response (201):
```json
{"id": "uuid", "name": "...", "status": "running", "community_id": "...", "prompt_id": "..."}
```

### GET /api/v1/agents

Response (200): Array of `AgentResponse`.

### DELETE /api/v1/agents/:id

Request (optional): `{"reason": "..."}`

Response: 204 No Content.

## Acceptance Criteria

1. POST with valid body returns 201 with agent response
2. POST with missing required field returns 400
3. GET returns list of agents
4. DELETE returns 204
5. Routes registered via `RegisterRoutes`

## Files

- `internal/keeper/adapters/inbound/httphandler/handler.go` ✅ IMPLEMENTED
- `internal/keeper/adapters/inbound/httphandler/handler_test.go` ✅ 5 tests passing
