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

// CommunityHandler implements the HTTP controllers for Communities CRUD operations.
type CommunityHandler struct {
	repo outbound.CommunityRepository
}

// NewCommunityHandler creates a new instance of CommunityHandler.
func NewCommunityHandler(repo outbound.CommunityRepository) *CommunityHandler {
	return &CommunityHandler{repo: repo}
}

// CreateCommunityRequest defines the request payload for creating a Community.
type CreateCommunityRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	Topology      string                 `json:"topology" binding:"required"`
	Configuration map[string]interface{} `json:"configuration"`
}

// UpdateCommunityRequest defines the request payload for updating a Community.
type UpdateCommunityRequest struct {
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	Topology      string                 `json:"topology" binding:"required"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        string                 `json:"status"`
}

// Create handles POST /api/v1/communities
func (h *CommunityHandler) Create(c *gin.Context) {
	var req CreateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	config := req.Configuration
	if config == nil {
		config = make(map[string]interface{})
	}

	comm := &domain.Community{
		ID:            uuid.New(),
		TenantID:      ten.FullName(),
		Name:          req.Name,
		Description:   req.Description,
		Topology:      domain.CommunityTopology(req.Topology),
		Configuration: config,
		Status:        domain.CommunityStatusCreated,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := comm.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), comm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comm)
}

// GetByID handles GET /api/v1/communities/:id
func (h *CommunityHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Param("community_id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	comm, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, comm)
}

// List handles GET /api/v1/communities
func (h *CommunityHandler) List(c *gin.Context) {
	comms, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, comms)
}

// Update handles PUT /api/v1/communities/:id
func (h *CommunityHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Param("community_id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var req UpdateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	config := req.Configuration
	if config == nil {
		config = make(map[string]interface{})
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Topology = domain.CommunityTopology(req.Topology)
	existing.Configuration = config
	if req.Status != "" {
		existing.Status = domain.CommunityStatus(req.Status)
	}
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

// Delete handles DELETE /api/v1/communities/:id
func (h *CommunityHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Param("community_id")
	}
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
