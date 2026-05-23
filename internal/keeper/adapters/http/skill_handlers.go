package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
)

// SkillHandler implements the HTTP controllers for Skills CRUD and relational operations.
type SkillHandler struct {
	repo outbound.SkillRepository
}

// NewSkillHandler creates a new instance of SkillHandler.
func NewSkillHandler(repo outbound.SkillRepository) *SkillHandler {
	return &SkillHandler{repo: repo}
}

// CreateSkillRequest defines the request payload for creating a Skill.
type CreateSkillRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	MCPServers   []string `json:"mcp_servers"`
	AllowedTools []string `json:"allowed_tools"`
	DeniedTools  []string `json:"denied_tools"`
}

// UpdateSkillRequest defines the request payload for updating a Skill.
type UpdateSkillRequest struct {
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	MCPServers   []string `json:"mcp_servers"`
	AllowedTools []string `json:"allowed_tools"`
	DeniedTools  []string `json:"denied_tools"`
}

// Create handles POST /api/v1/skills
func (h *SkillHandler) Create(c *gin.Context) {
	var req CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var mcpUUIDs []uuid.UUID
	for _, idStr := range req.MCPServers {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp_server id: " + idStr})
			return
		}
		mcpUUIDs = append(mcpUUIDs, id)
	}

	skill := &domain.Skill{
		ID:           uuid.New(),
		TenantID:     ten.FullName(),
		Name:         req.Name,
		Description:  req.Description,
		MCPServers:   mcpUUIDs,
		AllowedTools: req.AllowedTools,
		DeniedTools:  req.DeniedTools,
		Status:       domain.SkillStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := skill.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), skill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, skill)
}

// GetByID handles GET /api/v1/skills/:id
func (h *SkillHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	skill, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, skill)
}

// List handles GET /api/v1/skills
func (h *SkillHandler) List(c *gin.Context) {
	skills, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, skills)
}

// Update handles PUT /api/v1/skills/:id
func (h *SkillHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var req UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var mcpUUIDs []uuid.UUID
	for _, idStr := range req.MCPServers {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp_server id: " + idStr})
			return
		}
		mcpUUIDs = append(mcpUUIDs, id)
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.MCPServers = mcpUUIDs
	existing.AllowedTools = req.AllowedTools
	existing.DeniedTools = req.DeniedTools
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/skills/:id
func (h *SkillHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// AttachSkillToAgent handles POST /api/v1/agents/:agent_id/skills/:skill_id
func (h *SkillHandler) AttachSkillToAgent(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	skillIDStr := c.Param("skill_id")
	skillID, err := uuid.Parse(skillIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill_id"})
		return
	}

	if err := h.repo.AttachSkillToAgent(c.Request.Context(), agentID, skillID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "attached"})
}

// DetachSkillFromAgent handles DELETE /api/v1/agents/:agent_id/skills/:skill_id
func (h *SkillHandler) DetachSkillFromAgent(c *gin.Context) {
	agentIDStr := c.Param("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}

	skillIDStr := c.Param("skill_id")
	skillID, err := uuid.Parse(skillIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill_id"})
		return
	}

	if err := h.repo.DetachSkillFromAgent(c.Request.Context(), agentID, skillID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
