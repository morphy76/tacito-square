package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain"
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

	if err := h.repo.CreateTemplate(c.Request.Context(), pt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, pt)
}

// GetTemplateByID handles GET /api/v1/prompts/:id
func (h *PromptHandler) GetTemplateByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	pt, err := h.repo.GetTemplateByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pt)
}

// ListTemplates handles GET /api/v1/prompts
func (h *PromptHandler) ListTemplates(c *gin.Context) {
	templates, err := h.repo.ListTemplates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// UpdateTemplate handles PUT /api/v1/prompts/:id
func (h *PromptHandler) UpdateTemplate(c *gin.Context) {
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

	existing, err := h.repo.GetTemplateByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	latest, err := h.repo.GetLatestTemplateByName(c.Request.Context(), existing.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Immutable versioning: create a NEW record with bumped version
	newPT := &domain.PromptTemplate{
		ID:        uuid.New(),
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

	if err := h.repo.CreateTemplate(c.Request.Context(), newPT); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, newPT)
}

// DeleteTemplate handles DELETE /api/v1/prompts/:id
func (h *PromptHandler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := h.repo.DeleteTemplate(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateCollection handles POST /api/v1/prompt-collections
func (h *PromptHandler) CreateCollection(c *gin.Context) {
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

	if err := h.repo.CreateCollection(c.Request.Context(), col); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, col)
}

// GetCollectionByID handles GET /api/v1/prompt-collections/:id
func (h *PromptHandler) GetCollectionByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	col, err := h.repo.GetCollectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, col)
}

// ListCollections handles GET /api/v1/prompt-collections
func (h *PromptHandler) ListCollections(c *gin.Context) {
	collections, err := h.repo.ListCollections(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
}

// UpdateCollection handles PUT /api/v1/prompt-collections/:id
func (h *PromptHandler) UpdateCollection(c *gin.Context) {
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

	existing, err := h.repo.GetCollectionByID(c.Request.Context(), id)
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

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Templates = templateUUIDs
	existing.UpdatedAt = time.Now().UTC()

	if err := existing.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.UpdateCollection(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteCollection handles DELETE /api/v1/prompt-collections/:id
func (h *PromptHandler) DeleteCollection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	if err := h.repo.DeleteCollection(c.Request.Context(), id); err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ResolveCollection handles GET /api/v1/prompt-collections/:id/resolve
func (h *PromptHandler) ResolveCollection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	resolved, err := h.repo.ResolveCollectionPrompts(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resolved)
}
