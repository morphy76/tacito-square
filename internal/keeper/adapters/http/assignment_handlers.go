package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

// AssignmentHandler implements HTTP endpoints for Agent-Community Assignment lifecycle.
type AssignmentHandler struct {
	agentRepo      outbound.AgentRepository
	crdCoordinator outbound.CRDCoordinator
}

// NewAssignmentHandler creates a new instance of AssignmentHandler.
func NewAssignmentHandler(agentRepo outbound.AgentRepository, crdCoordinator outbound.CRDCoordinator) *AssignmentHandler {
	return &AssignmentHandler{
		agentRepo:      agentRepo,
		crdCoordinator: crdCoordinator,
	}
}

// Assign handles POST /api/v1/communities/:community_id/agents/:agent_id
func (h *AssignmentHandler) Assign(c *gin.Context) {
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

	err = h.agentRepo.AssignToCommunity(c.Request.Context(), agentID, communityID)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": errStr})
		return
	}

	// Trigger CRD submission coordinator hook
	if agent, err := h.agentRepo.GetByID(c.Request.Context(), agentID); err == nil {
		_ = h.crdCoordinator.SubmitAgentCRD(c.Request.Context(), agent)
	}

	c.JSON(http.StatusOK, gin.H{"status": "assigned"})
}

// Unassign handles DELETE /api/v1/communities/:community_id/agents/:agent_id
func (h *AssignmentHandler) Unassign(c *gin.Context) {
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

	// Fetch agent details before unassigning to supply for CRD teardown
	agent, fetchErr := h.agentRepo.GetByID(c.Request.Context(), agentID)

	err = h.agentRepo.UnassignFromCommunity(c.Request.Context(), agentID, communityID)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": errStr})
		return
	}

	// Trigger CRD teardown coordinator hook
	if fetchErr == nil {
		_ = h.crdCoordinator.TeardownAgentCRD(c.Request.Context(), agent)
	}

	c.Status(http.StatusNoContent)
}
