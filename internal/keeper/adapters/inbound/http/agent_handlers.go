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

// AgentHandler implements the HTTP controllers for Agent templates CRUD operations.
type AgentHandler struct {
	repo inbound.AgentUseCase
}

// NewAgentHandler creates a new instance of AgentHandler.
func NewAgentHandler(repo inbound.AgentUseCase) *AgentHandler {
	return &AgentHandler{repo: repo}
}

type CreateBrainRequest struct {
	LLMBindingID string   `json:"llm_binding_id" binding:"required,uuid"`
	Temperature  *float64 `json:"temperature" binding:"omitempty,gte=0.0,lte=2.0"`
	MaxTokens    *int     `json:"max_tokens" binding:"omitempty,gt=0"`
}

type CreateShortTermRequest struct {
	KeyNamespace string `json:"key_namespace"`
	TTLSeconds   int    `json:"ttl_seconds" binding:"required,gt=0"`
}

type CreateLongTermRequest struct {
	CollectionName  string `json:"collection_name" binding:"required"`
	VectorDimension int    `json:"vector_dimension" binding:"required,gt=0"`
}

type CreateMCPClient struct {
	ClientID     string            `json:"client_id" binding:"required,uuid"`
	CustomEnv    map[string]string `json:"custom_env"`
	CustomArgs   []string          `json:"custom_args"`
	AllowedTools []string          `json:"allowed_tools"`
}

type DeploymentRequest struct {
	Tier string `json:"tier"`
}

type CreateAgentRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Role            string                 `json:"role"`
	Brain           CreateBrainRequest     `json:"brain" binding:"required"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory" binding:"required"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory" binding:"required"`
	Skills          []string               `json:"skills"`
	PromptTemplate  string                 `json:"prompt_template"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients"`
	Deployment      DeploymentRequest      `json:"deployment"`
}

type UpdateAgentRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Role            string                 `json:"role"`
	Brain           CreateBrainRequest     `json:"brain" binding:"required"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory" binding:"required"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory" binding:"required"`
	Skills          []string               `json:"skills"`
	PromptTemplate  string                 `json:"prompt_template"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients"`
	Deployment      DeploymentRequest      `json:"deployment"`
}


// Create handles POST /api/v1/agents
func (h *AgentHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	ctx, span := otel.Tracer("keeper").Start(ctx, "http.create_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	llmBindingID, err := uuid.Parse(req.Brain.LLMBindingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid llm_binding_id uuid"})
		return
	}


	var prompts []uuid.UUID
	if req.PromptTemplate != "" {
		ptUUID, err := uuid.Parse(req.PromptTemplate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt_template uuid"})
			return
		}
		prompts = append(prompts, ptUUID)
	}

	var skillUUIDs []uuid.UUID
	for _, skStr := range req.Skills {
		skUUID, err := uuid.Parse(skStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill uuid"})
			return
		}
		skillUUIDs = append(skillUUIDs, skUUID)
	}

	var mcpClients []model.MCPClientConfig
	for _, mcpReq := range req.MCPClients {
		mcpUUID, err := uuid.Parse(mcpReq.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp client id uuid"})
			return
		}
		mcpClients = append(mcpClients, model.MCPClientConfig{
			ClientID:     mcpUUID,
			CustomEnv:    mcpReq.CustomEnv,
			CustomArgs:   mcpReq.CustomArgs,
			AllowedTools: mcpReq.AllowedTools,
		})
	}

	agent := &model.Agent{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        req.Name,
		Description: req.Description,
		Brain: model.BrainConfig{
			LLMBindingID: llmBindingID,
			Temperature:  req.Brain.Temperature,
			MaxTokens:    req.Brain.MaxTokens,
		},
		ShortTermMemory: model.ShortTermMemoryConfig{
			KeyNamespace: req.ShortTermMemory.KeyNamespace,
			TTLSeconds:   req.ShortTermMemory.TTLSeconds,
		},
		LongTermMemory: model.LongTermMemoryConfig{
			CollectionName:  req.LongTermMemory.CollectionName,
			VectorDimension: req.LongTermMemory.VectorDimension,
		},
		Skills:            skillUUIDs,
		Prompts:           prompts,
		PromptCollections: []uuid.UUID{},
		MCPClients:        mcpClients,
		Tier:              req.Deployment.Tier,
		Status:            model.AgentStatusDefined,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := agent.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(ctx, agent); err != nil {
		reqLogger.Error().Err(err).Msg("failed to create agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", agent.ID.String()).
		Msg("Agent template created successfully")

	c.Header("Location", "/api/v1/agents/"+agent.ID.String())
	c.Status(http.StatusCreated)
}

// GetByID handles GET /api/v1/agents/:id
func (h *AgentHandler) GetByID(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.get_agent", trace.WithSpanKind(trace.SpanKindServer))
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
	if idStr == "" {
		idStr = c.Param("agent_id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	agent, err := h.repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// List handles GET /api/v1/agents
func (h *AgentHandler) List(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.list_agents", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	agents, err := h.repo.List(ctx)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if agents == nil {
		agents = make([]*model.Agent, 0)
	}
	c.JSON(http.StatusOK, agents)
}

// Update handles PUT /api/v1/agents/:id
func (h *AgentHandler) Update(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.update_agent", trace.WithSpanKind(trace.SpanKindServer))
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
	if idStr == "" {
		idStr = c.Param("agent_id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	var req UpdateAgentRequest
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
	previousValue := model.Agent{
		ID:                existing.ID,
		TenantID:          existing.TenantID,
		Name:              existing.Name,
		Description:       existing.Description,
		Brain:             existing.Brain,
		ShortTermMemory:   existing.ShortTermMemory,
		LongTermMemory:    existing.LongTermMemory,
		Skills:            append([]uuid.UUID(nil), existing.Skills...),
		Prompts:           append([]uuid.UUID(nil), existing.Prompts...),
		PromptCollections: append([]uuid.UUID(nil), existing.PromptCollections...),
		MCPClients:        append([]model.MCPClientConfig(nil), existing.MCPClients...),
		Status:            existing.Status,
		CommunityID:       existing.CommunityID,
		CreatedAt:         existing.CreatedAt,
		UpdatedAt:         existing.UpdatedAt,
	}

	llmBindingID, err := uuid.Parse(req.Brain.LLMBindingID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid llm_binding_id uuid"})
		return
	}


	var prompts []uuid.UUID
	if req.PromptTemplate != "" {
		ptUUID, err := uuid.Parse(req.PromptTemplate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt_template uuid"})
			return
		}
		prompts = append(prompts, ptUUID)
	}

	var skillUUIDs []uuid.UUID
	for _, skStr := range req.Skills {
		skUUID, err := uuid.Parse(skStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill uuid"})
			return
		}
		skillUUIDs = append(skillUUIDs, skUUID)
	}

	var mcpClients []model.MCPClientConfig
	for _, mcpReq := range req.MCPClients {
		mcpUUID, err := uuid.Parse(mcpReq.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp client id uuid"})
			return
		}
		mcpClients = append(mcpClients, model.MCPClientConfig{
			ClientID:     mcpUUID,
			CustomEnv:    mcpReq.CustomEnv,
			CustomArgs:   mcpReq.CustomArgs,
			AllowedTools: mcpReq.AllowedTools,
		})
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Brain = model.BrainConfig{
		LLMBindingID: llmBindingID,
		Temperature:  req.Brain.Temperature,
		MaxTokens:    req.Brain.MaxTokens,
	}
	existing.ShortTermMemory = model.ShortTermMemoryConfig{
		KeyNamespace: req.ShortTermMemory.KeyNamespace,
		TTLSeconds:   req.ShortTermMemory.TTLSeconds,
	}
	existing.LongTermMemory = model.LongTermMemoryConfig{
		CollectionName:  req.LongTermMemory.CollectionName,
		VectorDimension: req.LongTermMemory.VectorDimension,
	}
	existing.Skills = skillUUIDs
	existing.Prompts = prompts
	existing.MCPClients = mcpClients
	existing.Tier = req.Deployment.Tier
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
		reqLogger.Error().Err(err).Msg("failed to update agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", existing.ID.String()).
		Msg("Agent template updated successfully")

	c.JSON(http.StatusOK, previousValue)
}

// Delete handles DELETE /api/v1/agents/:id
func (h *AgentHandler) Delete(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.delete_agent", trace.WithSpanKind(trace.SpanKindServer))
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
	if idStr == "" {
		idStr = c.Param("agent_id")
	}
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
		reqLogger.Error().Err(err).Msg("failed to delete agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Str("agent_id", id.String()).
		Msg("Agent template deleted successfully")

	c.Status(http.StatusNoContent)
}

// AttachPromptToAgent handles POST /api/v1/agents/:id/prompts/:prompt_id
func (h *AgentHandler) AttachPromptToAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.attach_prompt_to_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("agent_id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent uuid"})
		return
	}

	promptIDStr := c.Param("prompt_id")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt uuid"})
		return
	}

	if err := h.repo.AttachPromptToAgent(ctx, agentID, promptID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to attach prompt to agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// DetachPromptFromAgent handles DELETE /api/v1/agents/:id/prompts/:prompt_id
func (h *AgentHandler) DetachPromptFromAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.detach_prompt_from_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("agent_id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent uuid"})
		return
	}

	promptIDStr := c.Param("prompt_id")
	promptID, err := uuid.Parse(promptIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt uuid"})
		return
	}

	if err := h.repo.DetachPromptFromAgent(ctx, agentID, promptID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to detach prompt from agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// AttachCollectionToAgent handles POST /api/v1/agents/:id/prompt-collections/:collection_id
func (h *AgentHandler) AttachCollectionToAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.attach_collection_to_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("agent_id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent uuid"})
		return
	}

	colIDStr := c.Param("collection_id")
	colID, err := uuid.Parse(colIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection uuid"})
		return
	}

	if err := h.repo.AttachCollectionToAgent(ctx, agentID, colID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to attach prompt collection to agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// DetachCollectionFromAgent handles DELETE /api/v1/agents/:id/prompt-collections/:collection_id
func (h *AgentHandler) DetachCollectionFromAgent(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.detach_collection_from_agent", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("agent_id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent uuid"})
		return
	}

	colIDStr := c.Param("collection_id")
	colID, err := uuid.Parse(colIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid collection uuid"})
		return
	}

	if err := h.repo.DetachCollectionFromAgent(ctx, agentID, colID); err != nil {
		reqLogger.Error().Err(err).Msg("failed to detach prompt collection from agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// ResolveEffectivePrompts handles GET /api/v1/agents/:id/prompts
func (h *AgentHandler) ResolveEffectivePrompts(c *gin.Context) {
	ctx, span := otel.Tracer("keeper").Start(c.Request.Context(), "http.resolve_effective_prompts", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	idStr := c.Param("agent_id")
	if idStr == "" {
		idStr = c.Param("id")
	}
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent uuid"})
		return
	}

	_, err = h.repo.GetByID(ctx, agentID)
	if err != nil {
		reqLogger.Warn().Err(err).Str("agent_id", agentID.String()).Msg("agent not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	resolved, err := h.repo.ResolveEffectivePrompts(ctx, agentID)
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to resolve effective prompts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if resolved == nil {
		resolved = make([]*model.ResolvedAgentPrompt, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id":         agentID,
		"resolved_prompts": resolved,
		"total":            len(resolved),
	})
}
