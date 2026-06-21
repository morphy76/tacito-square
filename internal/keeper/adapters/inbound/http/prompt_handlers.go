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

// PromptHandler implements the HTTP controllers for Prompts and Collections.
type PromptHandler struct {
	repo inbound.PromptUseCase
}

// NewPromptHandler creates a new instance of PromptHandler.
func NewPromptHandler(repo inbound.PromptUseCase) *PromptHandler {
	return &PromptHandler{repo: repo}
}

// CreatePromptTemplateRequest defines the payload for creating a Prompt Template.
type CreatePromptTemplateRequest struct {
	Name    string              `json:"name" binding:"required"`
	Content string              `json:"content"`
	Status  model.PromptStatus `json:"status"`
}

// UpdatePromptTemplateRequest defines the payload for updating a Prompt Template.
type UpdatePromptTemplateRequest struct {
	Name    string              `json:"name" binding:"required"`
	Content string              `json:"content"`
	Status  model.PromptStatus `json:"status" binding:"required"`
}


// CreateTemplate handles POST /api/v1/prompts
func (h *PromptHandler) CreateTemplate(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := req.Status
	if status == "" {
		status = model.PromptStatusActive
	}

	pt := &model.PromptTemplate{
		ID:        uuid.New(),
		TenantID:  ten.FullName(),
		Name:      req.Name,
		Content:   req.Content,
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}

	if err := pt.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.CreateTemplate(ctx, pt); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create prompt template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_template_id", pt.ID.String()).
		Msg("Prompt template created successfully")

	c.Header("Location", "/api/v1/prompts/"+pt.ID.String())
	c.Status(http.StatusCreated)
}

// GetTemplateByID handles GET /api/v1/prompts/:id
func (h *PromptHandler) GetTemplateByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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

	pt, err := h.repo.GetTemplateByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pt)
}

// ListTemplates handles GET /api/v1/prompts
func (h *PromptHandler) ListTemplates(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_prompt_templates", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	templates, err := h.repo.ListTemplates(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list prompt templates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if templates == nil {
		templates = make([]*model.PromptTemplate, 0)
	}
	c.JSON(http.StatusOK, templates)
}

// UpdateTemplate handles PUT /api/v1/prompts/:id
func (h *PromptHandler) UpdateTemplate(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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

	if model.IsSystemLocked(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify system-locked prompt template"})
		return
	}

	var req UpdatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetTemplateByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Capture the unmodified state to return as the previous value
	previousValue := *existing

	// Update fields
	existing.Name = req.Name
	existing.Content = req.Content
	existing.Status = req.Status

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.UpdateTemplate(ctx, existing); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to update prompt template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_template_id", existing.ID.String()).
		Msg("Prompt template updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// DeleteTemplate handles DELETE /api/v1/prompts/:id
func (h *PromptHandler) DeleteTemplate(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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

	if model.IsSystemLocked(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify system-locked prompt template"})
		return
	}

	if err := h.repo.DeleteTemplate(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete prompt template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_template_id", id.String()).
		Msg("Prompt template deleted successfully")

	c.Status(http.StatusNoContent)
}

