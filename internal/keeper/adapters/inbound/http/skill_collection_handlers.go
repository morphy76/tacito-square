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

// CreateSkillCollectionRequest defines the payload for creating a Skill Collection.
type CreateSkillCollectionRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
}

// UpdateSkillCollectionRequest defines the payload for updating a Skill Collection.
type UpdateSkillCollectionRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
}

// CreateCollection handles POST /api/v1/skill-collections
func (h *SkillHandler) CreateCollection(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_skill_collection", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateSkillCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var skillUUIDs []uuid.UUID
	for _, idStr := range req.Skills {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id: " + idStr})
			return
		}
		skillUUIDs = append(skillUUIDs, id)
	}

	col := &model.SkillCollection{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        req.Name,
		Description: req.Description,
		Skills:      skillUUIDs,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := col.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.CreateCollection(ctx, col); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create skill collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_collection_id", col.ID.String()).
		Msg("Skill collection created successfully")

	c.Header("Location", "/api/v1/skill-collections/"+col.ID.String())
	c.Status(http.StatusCreated)
}

// GetCollectionByID handles GET /api/v1/skill-collections/:id
func (h *SkillHandler) GetCollectionByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_skill_collection", trace.WithSpanKind(trace.SpanKindServer))
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

// ListCollections handles GET /api/v1/skill-collections
func (h *SkillHandler) ListCollections(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_skill_collections", trace.WithSpanKind(trace.SpanKindServer))
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
		reqLogger.Error().Err(err).Msg("failed to list skill collections")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if collections == nil {
		collections = make([]*model.SkillCollection, 0)
	}
	c.JSON(http.StatusOK, collections)
}

// UpdateCollection handles PUT /api/v1/skill-collections/:id
func (h *SkillHandler) UpdateCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_skill_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdateSkillCollectionRequest
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

	// Capture previous state
	previousValue := model.SkillCollection{
		ID:          existing.ID,
		TenantID:    existing.TenantID,
		Name:        existing.Name,
		Description: existing.Description,
		Skills:      append([]uuid.UUID(nil), existing.Skills...),
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	}

	var skillUUIDs []uuid.UUID
	for _, idStr := range req.Skills {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill id: " + idStr})
			return
		}
		skillUUIDs = append(skillUUIDs, id)
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Skills = skillUUIDs
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
		reqLogger.Error().Err(err).Msg("failed to update skill collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_collection_id", existing.ID.String()).
		Msg("Skill collection updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// DeleteCollection handles DELETE /api/v1/skill-collections/:id
func (h *SkillHandler) DeleteCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_skill_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.DeleteCollection(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete skill collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("skill_collection_id", id.String()).
		Msg("Skill collection deleted successfully")

	c.Status(http.StatusNoContent)
}

// ResolveCollection handles GET /api/v1/skill-collections/:id/resolve
func (h *SkillHandler) ResolveCollection(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.resolve_skill_collection", trace.WithSpanKind(trace.SpanKindServer))
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

	resolved, err := h.repo.ResolveCollectionSkills(ctx, id)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to resolve skill collection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resolved == nil {
		resolved = make([]*model.Skill, 0)
	}
	c.JSON(http.StatusOK, resolved)
}
