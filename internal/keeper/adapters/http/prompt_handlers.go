package http

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// PromptHandler implements the HTTP controllers for Prompts and Collections.
type PromptHandler struct {
	repo outbound.PromptRepository
}

// NewPromptHandler creates a new instance of PromptHandler.
func NewPromptHandler(repo outbound.PromptRepository) *PromptHandler {
	return &PromptHandler{repo: repo}
}

// CreatePromptTemplateRequest defines the payload for creating a Prompt Template.
type CreatePromptTemplateRequest struct {
	Name    string              `json:"name" binding:"required"`
	Content string              `json:"content"`
	Role    domain.PromptRole   `json:"role" binding:"required"`
	Status  domain.PromptStatus `json:"status"`
}

// UpdatePromptTemplateRequest defines the payload for updating/versioning a Prompt Template.
type UpdatePromptTemplateRequest struct {
	Content string              `json:"content"`
	Role    domain.PromptRole   `json:"role" binding:"required"`
	Status  domain.PromptStatus `json:"status" binding:"required"`
}

// CreateCollectionRequest defines the payload for creating a Prompt Collection.
type CreateCollectionRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Templates   []string `json:"templates"`
}

// UpdateCollectionRequest defines the payload for updating a Prompt Collection.
type UpdateCollectionRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Templates   []string `json:"templates"`
}

// CreateTemplate handles POST /api/v1/prompts
func (h *PromptHandler) CreateTemplate(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.create_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

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
		status = domain.PromptStatusActive
	}

	pt := &domain.PromptTemplate{
		ID:        uuid.New(),
		TenantID:  ten.FullName(),
		Name:      req.Name,
		Content:   req.Content,
		Role:      req.Role,
		Version:   1,
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

	c.JSON(http.StatusCreated, pt)
}

// GetTemplateByID handles GET /api/v1/prompts/:id
func (h *PromptHandler) GetTemplateByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
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

	pt, err := h.repo.GetTemplateByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pt)
}

// ListTemplates handles GET /api/v1/prompts
func (h *PromptHandler) ListTemplates(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_prompt_templates", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

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
	c.JSON(http.StatusOK, templates)
}

// UpdateTemplate handles PUT /api/v1/prompts/:id
func (h *PromptHandler) UpdateTemplate(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetTemplateByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	latest, err := h.repo.GetLatestTemplateByName(ctx, existing.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Immutable versioning: create a NEW record with bumped version
	newPT := &domain.PromptTemplate{
		ID:        uuid.New(),
		TenantID:  ten.FullName(),
		Name:      existing.Name,
		Content:   req.Content,
		Role:      req.Role,
		Version:   latest.Version + 1,
		Status:    req.Status,
		CreatedAt: time.Now().UTC(),
	}

	if err := newPT.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.CreateTemplate(ctx, newPT); err != nil {
		reqLogger.Error().Err(err).Msg("failed to update prompt template version")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_template_id", newPT.ID.String()).
		Msg("Prompt template updated successfully")

	c.JSON(http.StatusOK, newPT)
}

// DeleteTemplate handles DELETE /api/v1/prompts/:id
func (h *PromptHandler) DeleteTemplate(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_prompt_template", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.DeleteTemplate(ctx, id); err != nil {
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

// CreateCollection handles POST /api/v1/prompt-collections
func (h *PromptHandler) CreateCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.create_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var templateUUIDs []uuid.UUID
	for _, idStr := range req.Templates {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id: " + idStr})
			return
		}
		templateUUIDs = append(templateUUIDs, id)
	}

	col := &domain.PromptCollection{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        req.Name,
		Description: req.Description,
		Templates:   templateUUIDs,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := col.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.CreateCollection(ctx, col); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_collection_id", col.ID.String()).
		Msg("Prompt collection template created successfully")

	c.JSON(http.StatusCreated, col)
}

// GetCollectionByID handles GET /api/v1/prompt-collections/:id
func (h *PromptHandler) GetCollectionByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	col, err := h.repo.GetCollectionByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, col)
}

// ListCollections handles GET /api/v1/prompt-collections
func (h *PromptHandler) ListCollections(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_prompt_collections", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	collections, err := h.repo.ListCollections(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list prompt collections")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
}

// UpdateCollection handles PUT /api/v1/prompt-collections/:id
func (h *PromptHandler) UpdateCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetCollectionByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var templateUUIDs []uuid.UUID
	for _, idStr := range req.Templates {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id: " + idStr})
			return
		}
		templateUUIDs = append(templateUUIDs, id)
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Templates = templateUUIDs
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.UpdateCollection(ctx, existing); err != nil {
		reqLogger.Error().Err(err).Msg("failed to update prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_collection_id", existing.ID.String()).
		Msg("Prompt collection template updated successfully")

	c.JSON(http.StatusOK, existing)
}

// DeleteCollection handles DELETE /api/v1/prompt-collections/:id
func (h *PromptHandler) DeleteCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.DeleteCollection(ctx, id); err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_collection_id", id.String()).
		Msg("Prompt collection template deleted successfully")

	c.Status(http.StatusNoContent)
}

// ResolveCollection handles GET /api/v1/prompt-collections/:id/resolve
func (h *PromptHandler) ResolveCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.resolve_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	resolved, err := h.repo.ResolveCollectionPrompts(ctx, id)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to resolve prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resolved)
}
