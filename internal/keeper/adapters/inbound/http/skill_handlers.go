package http

import (
	"context"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// SkillHandler implements the HTTP controllers for Skills CRUD, Skill Collections, and relational operations.
type SkillHandler struct {
	repo inbound.SkillUseCase
}

// NewSkillHandler creates a new instance of SkillHandler.
func NewSkillHandler(repo inbound.SkillUseCase) *SkillHandler {
	return &SkillHandler{repo: repo}
}

// parseUUID parses a path parameter to a UUID. If parsing fails, it responds with a 400 Bad Request and returns (uuid.Nil, false).
func (h *SkillHandler) parseUUID(c *gin.Context, paramName string) (uuid.UUID, bool) {
	valStr := c.Param(paramName)
	id, err := uuid.Parse(valStr)
	if err != nil {
		errMsg := "invalid " + paramName
		if paramName == "id" {
			errMsg = "invalid uuid"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return uuid.Nil, false
	}
	return id, true
}

// CreateSkillRequest defines the request payload for creating a Skill.
type CreateSkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// UpdateSkillRequest defines the request payload for updating a Skill.
type UpdateSkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// Create handles POST /api/v1/skills
func (h *SkillHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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

	skill := &model.Skill{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Status:      model.SkillStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
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
		Msg("Skill created successfully")

	c.Header("Location", "/api/v1/skills/"+skill.ID.String())
	c.Status(http.StatusCreated)
}

// GetByID handles GET /api/v1/skills/:id
func (h *SkillHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	id, ok := h.parseUUID(c, "id")
	if !ok {
		return
	}

	skill, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, skill)
}

// List handles GET /api/v1/skills
func (h *SkillHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_skills", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	id, ok := h.parseUUID(c, "id")
	if !ok {
		return
	}

	var req UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Capture previous state
	previousValue := *existing

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Content = req.Content
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Update(ctx, existing); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to update skill")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_id", existing.ID.String()).
		Msg("Skill updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// Delete handles DELETE /api/v1/skills/:id
func (h *SkillHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_skill", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	id, ok := h.parseUUID(c, "id")
	if !ok {
		return
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
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
		Msg("Skill deleted successfully")

	c.Status(http.StatusNoContent)
}

// AttachSkillToAgent handles POST /api/v1/agents/:agent_id/skills/:skill_id
func (h *SkillHandler) AttachSkillToAgent(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.attach_skill_to_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agentID, ok := h.parseUUID(c, "agent_id")
	if !ok {
		return
	}

	skillID, ok := h.parseUUID(c, "skill_id")
	if !ok {
		return
	}

	if err := h.repo.AttachSkillToAgent(ctx, agentID, skillID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
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
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agentID, ok := h.parseUUID(c, "agent_id")
	if !ok {
		return
	}

	skillID, ok := h.parseUUID(c, "skill_id")
	if !ok {
		return
	}

	if err := h.repo.DetachSkillFromAgent(ctx, agentID, skillID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
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


