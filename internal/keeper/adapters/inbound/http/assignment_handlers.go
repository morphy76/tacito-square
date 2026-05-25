package http

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// AssignmentHandler implements HTTP endpoints for Agent-Community Assignment lifecycle.
type AssignmentHandler struct {
	usecase inbound.AssignmentUseCase
}

// NewAssignmentHandler creates a new instance of AssignmentHandler.
func NewAssignmentHandler(usecase inbound.AssignmentUseCase) *AssignmentHandler {
	return &AssignmentHandler{
		usecase: usecase,
	}
}

// Assign handles POST /api/v1/communities/:community_id/agents/:agent_id
func (h *AssignmentHandler) Assign(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.assign_agent", trace.WithSpanKind(trace.SpanKindServer))
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
	communityID, err := uuid.Parse(commIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	err = h.usecase.Assign(ctx, communityID, agentID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": errStr})
			return
		}
		if strings.Contains(errStr, "already assigned") {
			c.JSON(http.StatusConflict, gin.H{"error": errStr})
			return
		}
		if strings.Contains(errStr, "status") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errStr})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to assign agent to community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": errStr})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", communityID.String()).
		Str("agent_id", agentID.String()).
		Msg("Agent successfully assigned to community")

	c.JSON(http.StatusOK, gin.H{"status": "assigned"})
}

// Unassign handles DELETE /api/v1/communities/:community_id/agents/:agent_id
func (h *AssignmentHandler) Unassign(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.unassign_agent", trace.WithSpanKind(trace.SpanKindServer))
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
	communityID, err := uuid.Parse(commIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid community_id"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	err = h.usecase.Unassign(ctx, communityID, agentID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": errStr})
			return
		}
		if strings.Contains(errStr, "not assigned") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errStr})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to unassign agent from community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": errStr})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", communityID.String()).
		Str("agent_id", agentID.String()).
		Msg("Agent successfully unassigned from community")

	c.Status(http.StatusNoContent)
}
