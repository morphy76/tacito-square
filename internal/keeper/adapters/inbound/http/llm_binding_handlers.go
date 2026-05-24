package http

import (
	"errors"
	"net/http"
	"os"
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

// LLMBindingHandler implements the HTTP controllers for LLM bindings CRUD operations.
type LLMBindingHandler struct {
	repo inbound.LLMBindingUseCase
}

// NewLLMBindingHandler creates a new instance of LLMBindingHandler.
func NewLLMBindingHandler(repo inbound.LLMBindingUseCase) *LLMBindingHandler {
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
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.create_llm_binding", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateLLMBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	binding := &model.LLMBinding{
		ID:                 uuid.New(),
		TenantID:           ten.FullName(),
		Name:               req.Name,
		Description:        req.Description,
		Provider:           model.Provider(req.Provider),
		APIBaseURL:         req.APIBaseURL,
		APIKeySecretRef:    req.APIKeySecretRef,
		DefaultModel:       req.DefaultModel,
		DefaultTemperature: temp,
		DefaultMaxTokens:   maxTokens,
		TimeoutSeconds:     timeout,
		Status:             model.StatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := binding.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(ctx, binding); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create llm binding")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("llm_binding_id", binding.ID.String()).
		Str("provider", string(binding.Provider)).
		Msg("LLM provider binding template created successfully")

	c.JSON(http.StatusCreated, binding)
}

// GetByID handles GET /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_llm_binding", trace.WithSpanKind(trace.SpanKindServer))
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

	binding, err := h.repo.GetByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, binding)
}

// List handles GET /api/v1/llm-bindings
func (h *LLMBindingHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_llm_bindings", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithTraceID(logger, span.SpanContext())

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	bindings, err := h.repo.List(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list llm bindings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindings)
}

// Update handles PUT /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_llm_binding", trace.WithSpanKind(trace.SpanKindServer))
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

	var req UpdateLLMBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.repo.GetByID(ctx, id)
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

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Provider = model.Provider(req.Provider)
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

	if err := h.repo.Update(ctx, existing); err != nil {
		reqLogger.Error().Err(err).Msg("failed to update llm binding")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("llm_binding_id", existing.ID.String()).
		Msg("LLM provider binding template updated successfully")

	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/v1/llm-bindings/:id
func (h *LLMBindingHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_llm_binding", trace.WithSpanKind(trace.SpanKindServer))
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

	if err := h.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		reqLogger.Error().Err(err).Msg("failed to delete llm binding")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("llm_binding_id", id.String()).
		Msg("LLM provider binding template deleted successfully")

	c.Status(http.StatusNoContent)
}
