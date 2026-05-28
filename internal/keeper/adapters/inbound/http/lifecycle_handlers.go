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

type LifecycleHandler struct {
	lifecycleUseCase inbound.LifecycleUseCase
}

func NewLifecycleHandler(lifecycleUseCase inbound.LifecycleUseCase) *LifecycleHandler {
	return &LifecycleHandler{lifecycleUseCase: lifecycleUseCase}
}

// DeployAgent handles POST /api/v1/agents/:agent_id/deploy
func (h *LifecycleHandler) DeployAgent(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.deploy_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	err = h.lifecycleUseCase.DeployAgent(ctx, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "agent not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "must be assigned to a community") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already pending or running") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to deploy agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", agentID.String()).
		Msg("Agent deployment triggered successfully")

	c.JSON(http.StatusAccepted, nil)
}

// UndeployAgent handles POST /api/v1/agents/:agent_id/undeploy
func (h *LifecycleHandler) UndeployAgent(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.undeploy_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	err = h.lifecycleUseCase.UndeployAgent(ctx, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "agent not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already undeployed/stopped") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to undeploy agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", agentID.String()).
		Msg("Agent undeployment completed successfully")

	c.JSON(http.StatusOK, nil)
}

// GetAgentStatus handles GET /api/v1/agents/:agent_id/status
func (h *LifecycleHandler) GetAgentStatus(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_agent_status", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	details, err := h.lifecycleUseCase.GetAgentStatus(ctx, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "agent not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to get agent status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}

// DeployCommunity handles POST /api/v1/communities/:community_id/deploy
func (h *LifecycleHandler) DeployCommunity(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.deploy_community", trace.WithSpanKind(trace.SpanKindServer))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	details, err := h.lifecycleUseCase.DeployCommunity(ctx, commID)
	if err != nil {
		if strings.Contains(err.Error(), "community not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to deploy community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", commID.String()).
		Str("status", details.Status).
		Msg("Community deployment processed")

	if details.Status == "partial_success" {
		c.JSON(http.StatusMultiStatus, details)
		return
	}

	c.JSON(http.StatusAccepted, details)
}

// UndeployCommunity handles POST /api/v1/communities/:community_id/undeploy
func (h *LifecycleHandler) UndeployCommunity(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.undeploy_community", trace.WithSpanKind(trace.SpanKindServer))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	details, err := h.lifecycleUseCase.UndeployCommunity(ctx, commID)
	if err != nil {
		if strings.Contains(err.Error(), "community not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to undeploy community")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", commID.String()).
		Str("status", details.Status).
		Msg("Community undeployment processed")

	if details.Status == "partial_success" {
		c.JSON(http.StatusMultiStatus, details)
		return
	}

	c.JSON(http.StatusOK, details)
}

// GetCommunityStatus handles GET /api/v1/communities/:community_id/status
func (h *LifecycleHandler) GetCommunityStatus(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_community_status", trace.WithSpanKind(trace.SpanKindServer))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	details, err := h.lifecycleUseCase.GetCommunityStatus(ctx, commID)
	if err != nil {
		if strings.Contains(err.Error(), "community not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to get community status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}
