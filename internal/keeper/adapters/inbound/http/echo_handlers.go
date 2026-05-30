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

	var timeouts int
	for _, res := range resp.Results {
		if res.Error == "timeout" {
			timeouts++
		}
	}
	span.SetAttributes(
		attribute.Int("agent_count", len(resp.Results)),
		attribute.Int("timeout_count", timeouts),
	)

	reqLogger.Info().
		Str("community_id", commID.String()).
		Str("tenant_id", ten.FullName()).
		Int("result_count", len(resp.Results)).
		Msg("community echo completed")

	c.JSON(http.StatusOK, resp)
}
