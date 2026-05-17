# SPEC-NFR-HTTP: HTTP Framework

| Field         | Value                              |
|---------------|------------------------------------|
| ID            | SPEC-NFR-HTTP                      |
| Status        | VERIFIED                           |
| Milestone     | M1                                 |
| Component     | keeper, agent, bff                 |

## Specification

1. All HTTP APIs MUST be built with **Gin** (`github.com/gin-gonic/gin`).
2. Request validation MUST use Gin binding tags (`binding:"required"`, etc.).
3. Error responses MUST use `gin.H{"error": message}` JSON format.
4. Route registration MUST be centralized via `RegisterRoutes(*gin.Engine)` per handler group.
5. Tests MUST use `gin.TestMode` and `httptest.NewRecorder` with `r.ServeHTTP(w, req)`.

## Acceptance Criteria

1. Keeper handler uses `gin.Context` for all endpoints
2. `ShouldBindJSON` validates required fields, returns 400 on failure
3. Path params extracted via `c.Param("id")`
4. All tests pass through Gin's ServeHTTP (full middleware stack)

## Files

- `internal/keeper/adapters/inbound/httphandler/handler.go` ✅ IMPLEMENTED
- `internal/keeper/adapters/inbound/httphandler/handler_test.go` ✅ 5 tests passing
