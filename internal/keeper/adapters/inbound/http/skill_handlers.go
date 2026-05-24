package http

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// SkillHandler implements the HTTP controllers for Skills CRUD and relational operations.
type SkillHandler struct {
	repo inbound.SkillUseCase
}

// NewSkillHandler creates a new instance of SkillHandler.
func NewSkillHandler(repo inbound.SkillUseCase) *SkillHandler {
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
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.create_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	skill := &model.Skill{
		ID:           uuid.New(),
		TenantID:     ten.FullName(),
		Name:         req.Name,
		Description:  req.Description,
		MCPServers:   mcpUUIDs,
		AllowedTools: req.AllowedTools,
		DeniedTools:  req.DeniedTools,
		Status:       model.SkillStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := skill.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(ctx, skill); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create skill")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_id", skill.ID.String()).
		Msg("Skill template created successfully")

	c.JSON(http.StatusCreated, skill)
}

// GetByID handles GET /api/v1/skills/:id
func (h *SkillHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	skill, err := h.repo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, skill)
}

// List handles GET /api/v1/skills
func (h *SkillHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_skills", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	skills, err := h.repo.List(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list skills")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if skills == nil {
		skills = make([]*model.Skill, 0)
	}
	c.JSON(http.StatusOK, skills)
}

// Update handles PUT /api/v1/skills/:id
func (h *SkillHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

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

	existing, err := h.repo.GetByID(ctx, id)
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

	if err := h.repo.Update(ctx, existing); err != nil {
		reqLogger.Error().Err(err).Msg("failed to update skill")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_id", existing.ID.String()).
		Msg("Skill template updated successfully")

	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/skills/:id
func (h *SkillHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete skill")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_id", id.String()).
		Msg("Skill template deleted successfully")

	c.Status(http.StatusNoContent)
}

// AttachSkillToAgent handles POST /api/v1/agents/:agent_id/skills/:skill_id
func (h *SkillHandler) AttachSkillToAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.attach_skill_to_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

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

	if err := h.repo.AttachSkillToAgent(ctx, agentID, skillID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to attach skill to agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", agentID.String()).
		Str("skill_id", skillID.String()).
		Msg("Skill successfully attached to agent")

	c.JSON(http.StatusOK, gin.H{"status": "attached"})
}

// DetachSkillFromAgent handles DELETE /api/v1/agents/:agent_id/skills/:skill_id
func (h *SkillHandler) DetachSkillFromAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.detach_skill_from_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

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

	if err := h.repo.DetachSkillFromAgent(ctx, agentID, skillID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to detach skill from agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", agentID.String()).
		Str("skill_id", skillID.String()).
		Msg("Skill successfully detached from agent")

	c.Status(http.StatusNoContent)
}
