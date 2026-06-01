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

// MCPClientHandler implements the HTTP controllers for MCP clients CRUD operations.
type MCPClientHandler struct {
	repo inbound.MCPClientUseCase
}

// NewMCPClientHandler creates a new instance of MCPClientHandler.
func NewMCPClientHandler(repo inbound.MCPClientUseCase) *MCPClientHandler {
	return &MCPClientHandler{repo: repo}
}

// CreateMCPClientRequest defines the request payload for creating an MCP client.
type CreateMCPClientRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	Transport     string            `json:"transport" binding:"required,oneof=stdio sse"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	URL           string            `json:"url"`
	AuthSecretRef string            `json:"auth_secret_ref"`
}

// UpdateMCPClientRequest defines the request payload for updating an MCP client.
type UpdateMCPClientRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description"`
	Transport     string            `json:"transport" binding:"required,oneof=stdio sse"`
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	URL           string            `json:"url"`
	AuthSecretRef string            `json:"auth_secret_ref"`
}

// Create handles POST /api/v1/mcp-clients
func (h *MCPClientHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_mcp_client", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateMCPClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := &model.MCPClient{
		ID:            uuid.New(),
		TenantID:      ten.FullName(),
		Name:          req.Name,
		Description:   req.Description,
		Transport:     model.Transport(req.Transport),
		Command:       req.Command,
		Args:          req.Args,
		Env:           req.Env,
		URL:           req.URL,
		AuthSecretRef: req.AuthSecretRef,
		Status:        model.MCPClientStatusActive,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := client.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(ctx, client); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create mcp client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("mcp_client_id", client.ID.String()).
		Msg("MCP client template created successfully")

	c.Header("Location", "/api/v1/mcp-clients/"+client.ID.String())
	c.Status(http.StatusCreated)
}

// GetByID handles GET /api/v1/mcp-clients/:id
func (h *MCPClientHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_mcp_client", trace.WithSpanKind(trace.SpanKindServer))
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

	client, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

// List handles GET /api/v1/mcp-clients
func (h *MCPClientHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_mcp_clients", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	clients, err := h.repo.List(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list mcp clients")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if clients == nil {
		clients = make([]*model.MCPClient, 0)
	}
	c.JSON(http.StatusOK, clients)
}

// Update handles PUT /api/v1/mcp-clients/:id
func (h *MCPClientHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_mcp_client", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdateMCPClientRequest
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
	previousValue := model.MCPClient{
		ID:            existing.ID,
		TenantID:      existing.TenantID,
		Name:          existing.Name,
		Description:   existing.Description,
		Transport:     existing.Transport,
		Command:       existing.Command,
		Args:          append([]string(nil), existing.Args...),
		Env:           existing.Env, // map
		URL:           existing.URL,
		AuthSecretRef: existing.AuthSecretRef,
		Status:        existing.Status,
		CreatedAt:     existing.CreatedAt,
		UpdatedAt:     existing.UpdatedAt,
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Transport = model.Transport(req.Transport)
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

	if err := h.repo.Update(ctx, existing); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to update mcp client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("mcp_client_id", existing.ID.String()).
		Msg("MCP client template updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// Delete handles DELETE /api/v1/mcp-clients/:id
func (h *MCPClientHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_mcp_client", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete mcp client")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("mcp_client_id", id.String()).
		Msg("MCP client template deleted successfully")

	c.Status(http.StatusNoContent)
}
