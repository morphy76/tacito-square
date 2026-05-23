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

// AgentHandler implements the HTTP controllers for Agent templates CRUD operations.
type AgentHandler struct {
	repo outbound.AgentRepository
}

// NewAgentHandler creates a new instance of AgentHandler.
func NewAgentHandler(repo outbound.AgentRepository) *AgentHandler {
	return &AgentHandler{repo: repo}
}

type CreateBrainRequest struct {
	Model             string   `json:"model" binding:"required"`
	Temperature       *float64 `json:"temperature" binding:"omitempty,gte=0.0,lte=2.0"`
	MaxTokens         int      `json:"max_tokens" binding:"omitempty,gt=0"`
	Endpoint          string   `json:"endpoint"`
	CredentialsSecret string   `json:"credentials_secret"`
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
	ServerID   string            `json:"server_id" binding:"required,uuid"`
	CustomEnv  map[string]string `json:"custom_env"`
	CustomArgs []string          `json:"custom_args"`
}

type CreateAgentRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Brain           CreateBrainRequest     `json:"brain" binding:"required"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory" binding:"required"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory" binding:"required"`
	Skills          []string               `json:"skills"`
	PromptTemplate  string                 `json:"prompt_template"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients"`
}

type UpdateAgentRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	Brain           CreateBrainRequest     `json:"brain" binding:"required"`
	ShortTermMemory CreateShortTermRequest `json:"short_term_memory" binding:"required"`
	LongTermMemory  CreateLongTermRequest  `json:"long_term_memory" binding:"required"`
	Skills          []string               `json:"skills"`
	PromptTemplate  string                 `json:"prompt_template"`
	MCPClients      []CreateMCPClient      `json:"mcp_clients"`
}

// Create handles POST /api/v1/agents
func (h *AgentHandler) Create(c *gin.Context) {
	var req CreateAgentRequest
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
	if req.Brain.Temperature != nil {
		temp = *req.Brain.Temperature
	}
	maxTokens := 2048
	if req.Brain.MaxTokens > 0 {
		maxTokens = req.Brain.MaxTokens
	}

	var promptTemplateUUID uuid.UUID
	if req.PromptTemplate != "" {
		ptUUID, err := uuid.Parse(req.PromptTemplate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt_template uuid"})
			return
		}
		promptTemplateUUID = ptUUID
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

	var mcpClients []domain.MCPClientConfig
	for _, mcpReq := range req.MCPClients {
		mcpUUID, err := uuid.Parse(mcpReq.ServerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp server id uuid"})
			return
		}
		mcpClients = append(mcpClients, domain.MCPClientConfig{
			ServerID:   mcpUUID,
			CustomEnv:  mcpReq.CustomEnv,
			CustomArgs: mcpReq.CustomArgs,
		})
	}

	agent := &domain.Agent{
		ID:          uuid.New(),
		TenantID:    ten.FullName(),
		Name:        req.Name,
		Description: req.Description,
		Brain: domain.BrainConfig{
			Model:             req.Brain.Model,
			Temperature:       temp,
			MaxTokens:         maxTokens,
			Endpoint:          req.Brain.Endpoint,
			CredentialsSecret: req.Brain.CredentialsSecret,
		},
		ShortTermMemory: domain.ShortTermMemoryConfig{
			KeyNamespace: req.ShortTermMemory.KeyNamespace,
			TTLSeconds:   req.ShortTermMemory.TTLSeconds,
		},
		LongTermMemory: domain.LongTermMemoryConfig{
			CollectionName:  req.LongTermMemory.CollectionName,
			VectorDimension: req.LongTermMemory.VectorDimension,
		},
		Skills:         skillUUIDs,
		PromptTemplate: promptTemplateUUID,
		MCPClients:     mcpClients,
		Status:         domain.AgentStatusDefined,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := agent.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Create(c.Request.Context(), agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agent)
}

// GetByID handles GET /api/v1/agents/:id
func (h *AgentHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Param("agent_id")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
		return
	}

	agent, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// List handles GET /api/v1/agents
func (h *AgentHandler) List(c *gin.Context) {
	agents, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agents)
}

// Update handles PUT /api/v1/agents/:id
func (h *AgentHandler) Update(c *gin.Context) {
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

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	temp := 0.7
	if req.Brain.Temperature != nil {
		temp = *req.Brain.Temperature
	}
	maxTokens := 2048
	if req.Brain.MaxTokens > 0 {
		maxTokens = req.Brain.MaxTokens
	}

	var promptTemplateUUID uuid.UUID
	if req.PromptTemplate != "" {
		ptUUID, err := uuid.Parse(req.PromptTemplate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prompt_template uuid"})
			return
		}
		promptTemplateUUID = ptUUID
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

	var mcpClients []domain.MCPClientConfig
	for _, mcpReq := range req.MCPClients {
		mcpUUID, err := uuid.Parse(mcpReq.ServerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mcp server id uuid"})
			return
		}
		mcpClients = append(mcpClients, domain.MCPClientConfig{
			ServerID:   mcpUUID,
			CustomEnv:  mcpReq.CustomEnv,
			CustomArgs: mcpReq.CustomArgs,
		})
	}

	ten := tenant.FromContext(c.Request.Context())
	if ten == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	existing.TenantID = ten.FullName()
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Brain = domain.BrainConfig{
		Model:             req.Brain.Model,
		Temperature:       temp,
		MaxTokens:         maxTokens,
		Endpoint:          req.Brain.Endpoint,
		CredentialsSecret: req.Brain.CredentialsSecret,
	}
	existing.ShortTermMemory = domain.ShortTermMemoryConfig{
		KeyNamespace: req.ShortTermMemory.KeyNamespace,
		TTLSeconds:   req.ShortTermMemory.TTLSeconds,
	}
	existing.LongTermMemory = domain.LongTermMemoryConfig{
		CollectionName:  req.LongTermMemory.CollectionName,
		VectorDimension: req.LongTermMemory.VectorDimension,
	}
	existing.Skills = skillUUIDs
	existing.PromptTemplate = promptTemplateUUID
	existing.MCPClients = mcpClients
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

// Delete handles DELETE /api/v1/agents/:id
func (h *AgentHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Param("agent_id")
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
