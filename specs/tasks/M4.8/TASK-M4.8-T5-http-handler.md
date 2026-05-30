# TASK-M4.8-T5: Echo HTTP Handler

| Field       | Value                                                            |
|-------------|------------------------------------------------------------------|
| Task ID     | TASK-M4.8-T5                                                     |
| Spec        | SPEC-FR-M4.8                                                     |
| Boundary    | Inbound HTTP Adapter — `internal/keeper/adapters/inbound/http`   |
| Status      | VERIFIED                                                         |
| Depends On  | TASK-M4.8-T4                                                     |

## Objective

Implement the Gin HTTP handler for `POST /api/v1/communities/:community_id/echo` and register it in the `v1` route group following the established handler pattern.

## Files

| File | Action |
|------|--------|
| `internal/keeper/adapters/inbound/http/echo_handlers.go` | NEW |
| `internal/keeper/adapters/inbound/http/echo_handlers_test.go` | NEW |

## RED Phase

Create `internal/keeper/adapters/inbound/http/echo_handlers_test.go`. Tests MUST run in `gin.TestMode` through `ServeHTTP`. Use a hand-rolled mock for `EchoUseCase`:

- `TestEchoCommunity_OK`: Mock returns a valid `CommunityEchoResponse` with 2 results. Assert `200 OK` with correct JSON body: `community_id`, `woke_community`, `results[]`.
- `TestEchoCommunity_EmptyMessage_BadRequest`: Mock returns `service.ErrEmptyMessage`. Assert `400 Bad Request` with `{"error": "message must not be empty after sanitization"}`.
- `TestEchoCommunity_InvalidJSON`: POST with `Content-Type: application/json` but body `{not valid}`. Assert `400 Bad Request` with `{"error": ...}`.
- `TestEchoCommunity_MissingMessageField`: POST `{}` (valid JSON, missing required field). Assert `400 Bad Request`.
- `TestEchoCommunity_CommunityNotFound`: Mock returns `service.ErrCommunityNotFound`. Assert `404 Not Found`.
- `TestEchoCommunity_NoRunningAgents`: Mock returns `service.ErrNoRunningAgents`. Assert `503 Service Unavailable`.
- `TestEchoCommunity_NATSUnavailable`: Mock returns `service.ErrBroadcasterUnavailable`. Assert `503 Service Unavailable` with `{"error": "NATS messaging is not available"}`.
- `TestEchoCommunity_InvalidCommunityID`: Non-UUID `community_id` in path. Assert `400 Bad Request`.
- `TestEchoCommunity_MissingTenant`: No tenant in context (bypass middleware). Assert `401 Unauthorized`.
- `TestEchoCommunity_InternalError`: Mock returns an unexpected error. Assert `500 Internal Server Error`.

Run `make test` — tests must fail (RED).

## GREEN Phase

Create `internal/keeper/adapters/inbound/http/echo_handlers.go`:

```go
package http

import (
    "context"
    "errors"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
    "github.com/morphy76/tacito-square/internal/keeper/application/service"
    "github.com/morphy76/tacito-square/internal/shared/observability"
    "github.com/morphy76/tacito-square/internal/shared/tenant"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type EchoHandler struct {
    echoUseCase inbound.EchoUseCase
}

func NewEchoHandler(echoUseCase inbound.EchoUseCase) *EchoHandler {
    return &EchoHandler{echoUseCase: echoUseCase}
}

type echoRequest struct {
    Message string `json:"message" binding:"required"`
}

// RegisterRoutes registers the echo route on the provided router group.
func (h *EchoHandler) RegisterRoutes(v1 *gin.RouterGroup) {
    v1.POST("/communities/:community_id/echo", h.EchoCommunity)
}

// EchoCommunity handles POST /api/v1/communities/:community_id/echo
// @Tags community/echo
func (h *EchoHandler) EchoCommunity(c *gin.Context) {
    ctx, cancel := context.WithCancel(c.Request.Context())
    defer cancel()

    ctx, span := otel.Tracer("keeper").Start(ctx, "http.echo_community",
        trace.WithSpanKind(trace.SpanKindServer))
    defer span.End()

    logger := observability.NewLogger("info", os.Stdout)
    reqLogger := observability.WithContext(logger, ctx)

    ten := tenant.FromContext(ctx)
    if ten == nil {
        reqLogger.Warn().Msg("unauthorized: missing tenant context")
        c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
        return
    }

    commIDStr := c.Param("community_id")
    commID, err := uuid.Parse(commIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id: must be a UUID"})
        return
    }

    var req echoRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    span.SetAttributes(
        attribute.String("community_id", commID.String()),
        attribute.String("tenant_id", ten.FullName()),
    )

    resp, err := h.echoUseCase.EchoCommunity(ctx, commID, req.Message)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrEmptyMessage):
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        case errors.Is(err, service.ErrCommunityNotFound):
            c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        case errors.Is(err, service.ErrNoRunningAgents),
            errors.Is(err, service.ErrBroadcasterUnavailable):
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
        default:
            reqLogger.Error().Err(err).Msg("echo community failed")
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        }
        return
    }

    reqLogger.Info().
        Str("community_id", commID.String()).
        Str("tenant_id", ten.FullName()).
        Int("result_count", len(resp.Results)).
        Msg("community echo completed")

    c.JSON(http.StatusOK, resp)
}
```

Run `make test` — tests must pass (GREEN).

## REFACTOR Phase

- Confirm the handler uses `c.ShouldBindJSON` (not `c.BindJSON`) to align with SPEC-NFR-HTTP.
- Confirm all error JSON payloads use the `"error"` key exactly.
- Confirm OTel span attributes include `agent_count` and `timeout_count` (add these by inspecting the response after the service call).
- Confirm the route is registered via `RegisterRoutes(*gin.RouterGroup)` — consistent with the project's route registration convention.
