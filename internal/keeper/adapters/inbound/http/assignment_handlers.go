package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

type assignRequest struct {
	AgentID uuid.UUID       `json:"agent_id" binding:"required"`
	Role    model.AgentRole `json:"role"`
}

type assignmentResponse struct {
	AgentID    uuid.UUID `json:"agent_id"`
	Role       string    `json:"role"`
	AssignedAt time.Time `json:"assigned_at"`
}

type assignmentListItem struct {
	AgentID    uuid.UUID  `json:"agent_id"`
	Role       string     `json:"role"`
	AssignedAt time.Time  `json:"assigned_at"`
	InformedAt *time.Time `json:"informed_at,omitempty"`
}

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

// Assign handles POST /api/v1/communities/:community_id/agents
func (h *AssignmentHandler) Assign(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	observability.StartHandlerSpan(c, "http.assign_agent")

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

	var req assignRequest
	_ = c.ShouldBindJSON(&req)
	if req.AgentID == uuid.Nil {
		if agentIDStr := c.Param("agent_id"); agentIDStr != "" {
			if parsed, parseErr := uuid.Parse(agentIDStr); parseErr == nil {
				req.AgentID = parsed
			}
		}
	}
	if req.AgentID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}

	err = h.usecase.Assign(ctx, communityID, req.AgentID, req.Role)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to assign agent to community")
		h.mapDomainErrorToHTTP(c, err, "assign agent")
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", communityID.String()).
		Str("agent_id", req.AgentID.String()).
		Msg("Agent successfully assigned to community")

	c.JSON(http.StatusCreated, assignmentResponse{
		AgentID:    req.AgentID,
		Role:       string(req.Role),
		AssignedAt: time.Now().UTC(),
	})
}

// Unassign handles DELETE /api/v1/communities/:community_id/agents/:agent_id
func (h *AssignmentHandler) Unassign(c *gin.Context) {
	ctx, _ := observability.StartHandlerSpan(c, "http.unassign_agent")

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
		reqLogger.Error().Err(err).Msg("failed to unassign agent from community")
		h.mapDomainErrorToHTTP(c, err, "unassign agent")
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("community_id", communityID.String()).
		Str("agent_id", agentID.String()).
		Msg("Agent successfully unassigned from community")

	c.Status(http.StatusNoContent)
}

// ListAssignments handles GET /api/v1/communities/:community_id/agents
func (h *AssignmentHandler) ListAssignments(c *gin.Context) {
	ctx, _ := observability.StartHandlerSpan(c, "http.list_assignments")

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

	list, err := h.usecase.ListByCommunity(ctx, communityID)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list community assignments")
		h.mapDomainErrorToHTTP(c, err, "list assignments")
		return
	}

	resp := make([]assignmentListItem, len(list))
	for i, item := range list {
		resp[i] = assignmentListItem{
			AgentID:    item.AgentID,
			Role:       string(item.Role),
			AssignedAt: item.AssignedAt,
			InformedAt: item.InformedAt,
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AssignmentHandler) mapDomainErrorToHTTP(c *gin.Context, err error, action string) {
	errStr := err.Error()
	if strings.Contains(errStr, "not found") {
		c.JSON(http.StatusNotFound, gin.H{"error": errStr})
		return
	}
	if strings.Contains(errStr, "already assigned") ||
		strings.Contains(errStr, "cannot have more than one") ||
		strings.Contains(errStr, "invalid role") {
		c.JSON(http.StatusConflict, gin.H{"error": errStr})
		return
	}
	if strings.Contains(errStr, "status") || strings.Contains(errStr, "not assigned") {
		c.JSON(http.StatusBadRequest, gin.H{"error": errStr})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to %s: %s", action, errStr)})
}
