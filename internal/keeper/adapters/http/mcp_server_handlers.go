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

// MCPServerHandler implements the HTTP controllers for MCP servers CRUD operations.
type MCPServerHandler struct {
	repo outbound.MCPServerRepository
}

// NewMCPServerHandler creates a new instance of MCPServerHandler.
func NewMCPServerHandler(repo outbound.MCPServerRepository) *MCPServerHandler {
	return &MCPServerHandler{repo: repo}
}

// CreateMCPServerRequest defines the request payload for creating an MCP server.
type CreateMCPServerRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	Transport     string            `json:"transport" binding:"required,oneof=stdio sse"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	URL           string            `json:"url"`
	AuthSecretRef string            `json:"auth_secret_ref"`
}

// UpdateMCPServerRequest defines the request payload for updating an MCP server.
type UpdateMCPServerRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	Transport     string            `json:"transport" binding:"required,oneof=stdio sse"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	URL           string            `json:"url"`
	AuthSecretRef string            `json:"auth_secret_ref"`
}

// Create handles POST /api/v1/mcp-servers
func (h *MCPServerHandler) Create(c *gin.Context) {
	var req CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	server := &domain.MCPServer{
		ID:            uuid.New(),
		TenantID:      ten.FullName(),
		Name:          req.Name,
		Description:   req.Description,
		Transport:     domain.Transport(req.Transport),
		Command:       req.Command,
		Args:          req.Args,
		Env:           req.Env,
		URL:           req.URL,
		AuthSecretRef: req.AuthSecretRef,
		Status:        domain.MCPServerStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := server.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

// GetByID handles GET /api/v1/mcp-servers/:id
func (h *MCPServerHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	server, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// List handles GET /api/v1/mcp-servers
func (h *MCPServerHandler) List(c *gin.Context) {
	servers, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

// Update handles PUT /api/v1/mcp-servers/:id
func (h *MCPServerHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var req UpdateMCPServerRequest
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

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Transport = domain.Transport(req.Transport)
	existing.Command = req.Command
	existing.Args = req.Args
	existing.Env = req.Env
	existing.URL = req.URL
	existing.AuthSecretRef = req.AuthSecretRef
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

// Delete handles DELETE /api/v1/mcp-servers/:id
func (h *MCPServerHandler) Delete(c *gin.Context) {
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
