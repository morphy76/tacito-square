package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

type ConfiguratorHandler struct {
	keeperClient outbound.KeeperClient
}

func NewConfiguratorHandler(keeperClient outbound.KeeperClient) *ConfiguratorHandler {
	return &ConfiguratorHandler{
		keeperClient: keeperClient,
	}
}

// GetWizardOptions fetches LLM bindings, skills, and prompts in parallel.
func (h *ConfiguratorHandler) GetWizardOptions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	type llmResult struct {
		bindings []*outbound.LLMBinding
		err      error
	}
	type skillsResult struct {
		skills []*outbound.Skill
		err    error
	}
	type promptsResult struct {
		prompts []*outbound.PromptTemplate
		err     error
	}

	llmChan := make(chan llmResult, 1)
	skillsChan := make(chan skillsResult, 1)
	promptsChan := make(chan promptsResult, 1)

	go func() {
		bindings, err := h.keeperClient.LLMBindings().List(ctx)
		llmChan <- llmResult{bindings: bindings, err: err}
	}()

	go func() {
		skills, err := h.keeperClient.Skills().List(ctx)
		skillsChan <- skillsResult{skills: skills, err: err}
	}()

	go func() {
		prompts, err := h.keeperClient.Prompts().List(ctx)
		promptsChan <- promptsResult{prompts: prompts, err: err}
	}()

	var bindings []*outbound.LLMBinding
	var skills []*outbound.Skill
	var prompts []*outbound.PromptTemplate
	var err error

	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "downstream request timed out"})
			return
		case r := <-llmChan:
			if r.err != nil {
				err = r.err
			}
			bindings = r.bindings
		case r := <-skillsChan:
			if r.err != nil {
				err = r.err
			}
			skills = r.skills
		case r := <-promptsChan:
			if r.err != nil {
				err = r.err
			}
			prompts = r.prompts
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Default empty slices to avoid null in JSON
	if bindings == nil {
		bindings = []*outbound.LLMBinding{}
	}
	if skills == nil {
		skills = []*outbound.Skill{}
	}
	if prompts == nil {
		prompts = []*outbound.PromptTemplate{}
	}

	c.JSON(http.StatusOK, gin.H{
		"llm_bindings": bindings,
		"skills":       skills,
		"prompts":      prompts,
	})
}

// Agents CRUD

func (h *ConfiguratorHandler) ListAgents(c *gin.Context) {
	ctx := c.Request.Context()
	agents, err := h.keeperClient.Agents().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if agents == nil {
		agents = []*outbound.Agent{}
	}
	c.JSON(http.StatusOK, agents)
}

func (h *ConfiguratorHandler) GetAgent(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	agent, err := h.keeperClient.Agents().Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (h *ConfiguratorHandler) CreateAgent(c *gin.Context) {
	ctx := c.Request.Context()
	var req outbound.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agent, err := h.keeperClient.Agents().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, agent)
}

func (h *ConfiguratorHandler) UpdateAgent(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	var req outbound.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agent, err := h.keeperClient.Agents().Update(ctx, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (h *ConfiguratorHandler) DeleteAgent(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	err = h.keeperClient.Agents().Delete(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Communities CRUD

func (h *ConfiguratorHandler) ListCommunities(c *gin.Context) {
	ctx := c.Request.Context()
	comms, err := h.keeperClient.Communities().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agents, err := h.keeperClient.Agents().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	commAgents := make(map[uuid.UUID][]string)
	for _, a := range agents {
		if a.CommunityID != nil {
			commAgents[*a.CommunityID] = append(commAgents[*a.CommunityID], a.ID.String())
		}
	}

	var resp []gin.H
	for _, comm := range comms {
		agentIDs := commAgents[comm.ID]
		if agentIDs == nil {
			agentIDs = []string{}
		}
		resp = append(resp, enrichCommunity(comm, agentIDs))
	}
	if resp == nil {
		resp = []gin.H{}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ConfiguratorHandler) GetCommunity(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	comm, err := h.keeperClient.Communities().Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	agents, err := h.keeperClient.Agents().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var agentIDs []string
	for _, a := range agents {
		if a.CommunityID != nil && *a.CommunityID == id {
			agentIDs = append(agentIDs, a.ID.String())
		}
	}
	if agentIDs == nil {
		agentIDs = []string{}
	}
	c.JSON(http.StatusOK, enrichCommunity(comm, agentIDs))
}

func (h *ConfiguratorHandler) CreateCommunity(c *gin.Context) {
	ctx := c.Request.Context()
	var req outbound.CreateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comm, err := h.keeperClient.Communities().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, enrichCommunity(comm, []string{}))
}

func (h *ConfiguratorHandler) UpdateCommunity(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	var req outbound.UpdateCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comm, err := h.keeperClient.Communities().Update(ctx, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fetch current assigned agents to enrich response
	agents, err := h.keeperClient.Agents().List(ctx)
	var agentIDs []string
	if err == nil {
		for _, a := range agents {
			if a.CommunityID != nil && *a.CommunityID == id {
				agentIDs = append(agentIDs, a.ID.String())
			}
		}
	}
	if agentIDs == nil {
		agentIDs = []string{}
	}

	c.JSON(http.StatusOK, enrichCommunity(comm, agentIDs))
}

func (h *ConfiguratorHandler) DeleteCommunity(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	err = h.keeperClient.Communities().Delete(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Assignments CRUD (Independent API operations)

func (h *ConfiguratorHandler) AssignAgent(c *gin.Context) {
	ctx := c.Request.Context()
	commID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id format"})
		return
	}

	err = h.keeperClient.Communities().AssignAgent(ctx, commID, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ConfiguratorHandler) UnassignAgent(c *gin.Context) {
	ctx := c.Request.Context()
	commID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}
	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id format"})
		return
	}

	err = h.keeperClient.Communities().UnassignAgent(ctx, commID, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// Advanced Sync

func (h *ConfiguratorHandler) AdvancedSync(c *gin.Context) {
	ctx := c.Request.Context()

	var rawItems []map[string]interface{}
	if err := c.ShouldBindJSON(&rawItems); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(rawItems) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
		return
	}

	firstItem := rawItems[0]
	isAgent := false
	if _, ok := firstItem["brain"]; ok {
		isAgent = true
	} else if _, ok := firstItem["short_term_memory"]; ok {
		isAgent = true
	}

	dataBytes, err := json.Marshal(rawItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize items"})
		return
	}

	if isAgent {
		var reqAgents []*outbound.Agent
		if err := json.Unmarshal(dataBytes, &reqAgents); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		currentAgents, err := h.keeperClient.Agents().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		currentMap := make(map[uuid.UUID]*outbound.Agent)
		for _, a := range currentAgents {
			currentMap[a.ID] = a
		}

		processedIDs := make(map[uuid.UUID]bool)

		for _, a := range reqAgents {
			if a.ID != uuid.Nil && currentMap[a.ID] != nil {
				updateReq := outbound.UpdateAgentRequest{
					Name:        a.Name,
					Description: a.Description,
					Role:        a.Role,
					Brain: outbound.CreateBrainRequest{
						LLMBindingID: a.Brain.LLMBindingID.String(),
						Temperature:  a.Brain.Temperature,
						MaxTokens:    a.Brain.MaxTokens,
					},
					ShortTermMemory: outbound.CreateShortTermRequest{
						KeyNamespace: a.ShortTermMemory.KeyNamespace,
						TTLSeconds:   a.ShortTermMemory.TTLSeconds,
					},
					LongTermMemory: outbound.CreateLongTermRequest{
						CollectionName:  a.LongTermMemory.CollectionName,
						VectorDimension: a.LongTermMemory.VectorDimension,
					},
				}
				for _, sID := range a.Skills {
					updateReq.Skills = append(updateReq.Skills, sID.String())
				}
				if a.PromptTemplate != uuid.Nil {
					updateReq.PromptTemplate = a.PromptTemplate.String()
				}
				for _, client := range a.MCPClients {
					updateReq.MCPClients = append(updateReq.MCPClients, outbound.CreateMCPClient{
						ClientID:     client.ClientID.String(),
						CustomEnv:    client.CustomEnv,
						CustomArgs:   client.CustomArgs,
						AllowedTools: client.AllowedTools,
					})
				}
				updateReq.Deployment = outbound.DeploymentRequest{Tier: a.Tier}

				_, err := h.keeperClient.Agents().Update(ctx, a.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[a.ID] = true
			} else {
				createReq := outbound.CreateAgentRequest{
					Name:        a.Name,
					Description: a.Description,
					Role:        a.Role,
					Brain: outbound.CreateBrainRequest{
						LLMBindingID: a.Brain.LLMBindingID.String(),
						Temperature:  a.Brain.Temperature,
						MaxTokens:    a.Brain.MaxTokens,
					},
					ShortTermMemory: outbound.CreateShortTermRequest{
						KeyNamespace: a.ShortTermMemory.KeyNamespace,
						TTLSeconds:   a.ShortTermMemory.TTLSeconds,
					},
					LongTermMemory: outbound.CreateLongTermRequest{
						CollectionName:  a.LongTermMemory.CollectionName,
						VectorDimension: a.LongTermMemory.VectorDimension,
					},
				}
				for _, sID := range a.Skills {
					createReq.Skills = append(createReq.Skills, sID.String())
				}
				if a.PromptTemplate != uuid.Nil {
					createReq.PromptTemplate = a.PromptTemplate.String()
				}
				for _, client := range a.MCPClients {
					createReq.MCPClients = append(createReq.MCPClients, outbound.CreateMCPClient{
						ClientID:     client.ClientID.String(),
						CustomEnv:    client.CustomEnv,
						CustomArgs:   client.CustomArgs,
						AllowedTools: client.AllowedTools,
					})
				}
				createReq.Deployment = outbound.DeploymentRequest{Tier: a.Tier}

				newAgent, err := h.keeperClient.Agents().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newAgent.ID] = true
			}
		}

		for _, a := range currentAgents {
			if !processedIDs[a.ID] {
				err := h.keeperClient.Agents().Delete(ctx, a.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	} else {
		var reqComms []*outbound.Community
		if err := json.Unmarshal(dataBytes, &reqComms); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		currentComms, err := h.keeperClient.Communities().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		currentMap := make(map[uuid.UUID]*outbound.Community)
		for _, c := range currentComms {
			currentMap[c.ID] = c
		}

		processedIDs := make(map[uuid.UUID]bool)

		for _, comm := range reqComms {
			if comm.ID != uuid.Nil && currentMap[comm.ID] != nil {
				updateReq := outbound.UpdateCommunityRequest{
					Name:          comm.Name,
					Description:   comm.Description,
					Topology:      comm.Topology,
					Configuration: comm.Configuration,
					Status:        comm.Status,
				}
				_, err := h.keeperClient.Communities().Update(ctx, comm.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[comm.ID] = true
			} else {
				createReq := outbound.CreateCommunityRequest{
					Name:          comm.Name,
					Description:   comm.Description,
					Topology:      comm.Topology,
					Configuration: comm.Configuration,
				}
				newComm, err := h.keeperClient.Communities().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newComm.ID] = true
			}
		}

		for _, comm := range currentComms {
			if !processedIDs[comm.ID] {
				err := h.keeperClient.Communities().Delete(ctx, comm.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func enrichCommunity(comm *outbound.Community, agents []string) gin.H {
	return gin.H{
		"id":            comm.ID,
		"tenant_id":     comm.TenantID,
		"name":          comm.Name,
		"description":   comm.Description,
		"topology":      comm.Topology,
		"configuration": comm.Configuration,
		"status":        comm.Status,
		"created_at":    comm.CreatedAt,
		"updated_at":    comm.UpdatedAt,
		"agents":        agents,
	}
}
