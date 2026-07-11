package http

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

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

// CreateCollection handles POST /api/v1/prompt-collections
func (h *PromptHandler) CreateCollection(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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

	col := &model.PromptCollection{
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
		Msg("Prompt collection created successfully")

	c.Header("Location", "/api/v1/prompt-collections/"+col.ID.String())
	c.Status(http.StatusCreated)
}

// GetCollectionByID handles GET /api/v1/prompt-collections/:id
func (h *PromptHandler) GetCollectionByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	col, err := h.repo.GetCollectionByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, col)
}

// ListCollections handles GET /api/v1/prompt-collections
func (h *PromptHandler) ListCollections(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_prompt_collections", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

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
	if collections == nil {
		collections = make([]*model.PromptCollection, 0)
	}
	c.JSON(http.StatusOK, collections)
}

// UpdateCollection handles PUT /api/v1/prompt-collections/:id
func (h *PromptHandler) UpdateCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetCollectionByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Capture the unmodified state as the previous value
	previousValue := model.PromptCollection{
		ID:          existing.ID,
		TenantID:    existing.TenantID,
		Name:        existing.Name,
		Description: existing.Description,
		Templates:   append([]uuid.UUID(nil), existing.Templates...),
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
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
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to update prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("prompt_collection_id", existing.ID.String()).
		Msg("Prompt collection updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// DeleteCollection handles DELETE /api/v1/prompt-collections/:id
func (h *PromptHandler) DeleteCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.DeleteCollection(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
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
		Msg("Prompt collection deleted successfully")

	c.Status(http.StatusNoContent)
}

// ResolveCollection handles GET /api/v1/prompt-collections/:id/resolve
func (h *PromptHandler) ResolveCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.resolve_prompt_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	resolved, err := h.repo.ResolveCollectionPrompts(ctx, id)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to resolve prompt collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resolved == nil {
		resolved = make([]*model.PromptTemplate, 0)
	}
	c.JSON(http.StatusOK, resolved)
}

// AddPromptToCollection handles POST /api/v1/prompt-collections/:id/prompts/:prompt_id
func (h *PromptHandler) AddPromptToCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.add_prompt_to_collection", trace.WithSpanKind(trace.SpanKindServer))
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
	colID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection uuid"})
		return
	}

	promptIDStr := c.Param("prompt_id")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt uuid"})
		return
	}

	if err := h.repo.AddPromptToCollection(ctx, colID, promptID); err != nil {
		if strings.Contains(err.Error(), "409 Conflict") || strings.Contains(err.Error(), "already in collection") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to add prompt to collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// RemovePromptFromCollection handles DELETE /api/v1/prompt-collections/:id/prompts/:prompt_id
func (h *PromptHandler) RemovePromptFromCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.remove_prompt_from_collection", trace.WithSpanKind(trace.SpanKindServer))
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
	colID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection uuid"})
		return
	}

	promptIDStr := c.Param("prompt_id")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt uuid"})
		return
	}

	if err := h.repo.RemovePromptFromCollection(ctx, colID, promptID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to remove prompt from collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
