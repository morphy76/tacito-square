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

// LLMBindingHandler implements the HTTP controllers for LLM bindings CRUD operations.
type LLMBindingHandler struct {
	repo outbound.LLMBindingRepository
}

// NewLLMBindingHandler creates a new instance of LLMBindingHandler.
func NewLLMBindingHandler(repo outbound.LLMBindingRepository) *LLMBindingHandler {
	return &LLMBindingHandler{repo: repo}
}

type CreateLLMBindingRequest struct {
	Name               string   `json:"name" binding:"required"`
	Description        string   `json:"description"`
	Provider           string   `json:"provider" binding:"required,oneof=openai anthropic groq ollama custom"`
	APIBaseURL         string   `json:"api_base_url" binding:"required,url"`
	APIKeySecretRef    string   `json:"api_key_secret_ref" binding:"required"`
	DefaultModel       string   `json:"default_model" binding:"required"`
	DefaultTemperature *float64 `json:"default_temperature" binding:"omitempty,gte=0.0,lte=2.0"`
	DefaultMaxTokens   int      `json:"default_max_tokens" binding:"omitempty,gt=0"`
	TimeoutSeconds     int      `json:"timeout_seconds" binding:"omitempty,gt=0"`
}

type UpdateLLMBindingRequest struct {
	Name               string   `json:"name" binding:"required"`
	Description        string   `json:"description"`
	Provider           string   `json:"provider" binding:"required,oneof=openai anthropic groq ollama custom"`
	APIBaseURL         string   `json:"api_base_url" binding:"required,url"`
	APIKeySecretRef    string   `json:"api_key_secret_ref" binding:"required"`
	DefaultModel       string   `json:"default_model" binding:"required"`
	DefaultTemperature *float64 `json:"default_temperature" binding:"omitempty,gte=0.0,lte=2.0"`
	DefaultMaxTokens   int      `json:"default_max_tokens" binding:"omitempty,gt=0"`
	TimeoutSeconds     int      `json:"timeout_seconds" binding:"omitempty,gt=0"`
}

// Create handles POST /api/v1/llm-bindings
func (h *LLMBindingHandler) Create(c *gin.Context) {
	var req CreateLLMBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	temp := 0.7
	if req.DefaultTemperature != nil {
		temp = *req.DefaultTemperature
	}
	maxTokens := 2048
	if req.DefaultMaxTokens > 0 {
		maxTokens = req.DefaultMaxTokens
	}
	timeout := 30
	if req.TimeoutSeconds > 0 {
		timeout = req.TimeoutSeconds
	}

	binding := &domain.LLMBinding{
		ID:                 uuid.New(),
		TenantID:           ten.FullName(),
		Name:               req.Name,
		Description:        req.Description,
		Provider:           domain.Provider(req.Provider),
		APIBaseURL:         req.APIBaseURL,
		APIKeySecretRef:    req.APIKeySecretRef,
		DefaultModel:       req.DefaultModel,
		DefaultTemperature: temp,
		DefaultMaxTokens:   maxTokens,
		TimeoutSeconds:     timeout,
		Status:             domain.StatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := binding.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), binding); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, binding)
}

// GetByID handles GET /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	binding, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, binding)
}

// List handles GET /api/v1/llm-bindings
func (h *LLMBindingHandler) List(c *gin.Context) {
	bindings, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindings)
}

// Update handles PUT /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var req UpdateLLMBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	temp := 0.7
	if req.DefaultTemperature != nil {
		temp = *req.DefaultTemperature
	}
	maxTokens := 2048
	if req.DefaultMaxTokens > 0 {
		maxTokens = req.DefaultMaxTokens
	}
	timeout := 30
	if req.TimeoutSeconds > 0 {
		timeout = req.TimeoutSeconds
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Provider = domain.Provider(req.Provider)
	existing.APIBaseURL = req.APIBaseURL
	existing.APIKeySecretRef = req.APIKeySecretRef
	existing.DefaultModel = req.DefaultModel
	existing.DefaultTemperature = temp
	existing.DefaultMaxTokens = maxTokens
	existing.TimeoutSeconds = timeout
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

// Delete handles DELETE /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) Delete(c *gin.Context) {
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
