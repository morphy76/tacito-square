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

// GetWizardOptions fetches LLM bindings, skills, prompts, and MCP servers in parallel.
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
	type mcpResult struct {
		servers []*outbound.MCPServer
		err     error
	}

	llmChan := make(chan llmResult, 1)
	skillsChan := make(chan skillsResult, 1)
	promptsChan := make(chan promptsResult, 1)
	mcpChan := make(chan mcpResult, 1)

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

	go func() {
		servers, err := h.keeperClient.MCPServers().List(ctx)
		mcpChan <- mcpResult{servers: servers, err: err}
	}()

	var bindings []*outbound.LLMBinding
	var skills []*outbound.Skill
	var prompts []*outbound.PromptTemplate
	var mcpServers []*outbound.MCPServer
	var err error

	for i := 0; i < 4; i++ {
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
		case r := <-mcpChan:
			if r.err != nil {
				err = r.err
			}
			mcpServers = r.servers
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
	if mcpServers == nil {
		mcpServers = []*outbound.MCPServer{}
	}

	c.JSON(http.StatusOK, gin.H{
		"llm_bindings": bindings,
		"skills":       skills,
		"prompts":      prompts,
		"mcp_servers":  mcpServers,
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
	isLLMBinding := false
	isMCPServer := false
	isSkill := false
	isPrompt := false

	if _, ok := firstItem["brain"]; ok {
		isAgent = true
	} else if _, ok := firstItem["short_term_memory"]; ok {
		isAgent = true
	} else if _, ok := firstItem["provider"]; ok {
		isLLMBinding = true
	} else if _, ok := firstItem["api_base_url"]; ok {
		isLLMBinding = true
	} else if _, ok := firstItem["transport"]; ok {
		isMCPServer = true
	} else if _, ok := firstItem["command"]; ok {
		isMCPServer = true
	} else if _, ok := firstItem["content"]; ok {
		if _, hasDesc := firstItem["description"]; hasDesc {
			isSkill = true
		} else {
			isPrompt = true
		}
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
	} else if isLLMBinding {
		var reqBindings []*outbound.LLMBinding
		if err := json.Unmarshal(dataBytes, &reqBindings); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		currentBindings, err := h.keeperClient.LLMBindings().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		currentMap := make(map[uuid.UUID]*outbound.LLMBinding)
		for _, b := range currentBindings {
			currentMap[b.ID] = b
		}
		processedIDs := make(map[uuid.UUID]bool)
		for _, b := range reqBindings {
			if b.ID != uuid.Nil && currentMap[b.ID] != nil {
				updateReq := outbound.UpdateLLMBindingRequest{
					Name:               b.Name,
					Description:        b.Description,
					Provider:           b.Provider,
					APIBaseURL:         b.APIBaseURL,
					APIKeySecretRef:    b.APIKeySecretRef,
					DefaultModel:       b.DefaultModel,
					DefaultTemperature: &b.DefaultTemperature,
					DefaultMaxTokens:   b.DefaultMaxTokens,
					TimeoutSeconds:     b.TimeoutSeconds,
				}
				_, err := h.keeperClient.LLMBindings().Update(ctx, b.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[b.ID] = true
			} else {
				createReq := outbound.CreateLLMBindingRequest{
					Name:               b.Name,
					Description:        b.Description,
					Provider:           b.Provider,
					APIBaseURL:         b.APIBaseURL,
					APIKeySecretRef:    b.APIKeySecretRef,
					DefaultModel:       b.DefaultModel,
					DefaultTemperature: &b.DefaultTemperature,
					DefaultMaxTokens:   b.DefaultMaxTokens,
					TimeoutSeconds:     b.TimeoutSeconds,
				}
				newBinding, err := h.keeperClient.LLMBindings().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newBinding.ID] = true
			}
		}
		for _, b := range currentBindings {
			if !processedIDs[b.ID] {
				err := h.keeperClient.LLMBindings().Delete(ctx, b.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	} else if isMCPServer {
		var reqServers []*outbound.MCPServer
		if err := json.Unmarshal(dataBytes, &reqServers); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		currentServers, err := h.keeperClient.MCPServers().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		currentMap := make(map[uuid.UUID]*outbound.MCPServer)
		for _, s := range currentServers {
			currentMap[s.ID] = s
		}
		processedIDs := make(map[uuid.UUID]bool)
		for _, s := range reqServers {
			if s.ID != uuid.Nil && currentMap[s.ID] != nil {
				updateReq := outbound.UpdateMCPServerRequest{
					Name:          s.Name,
					Description:   s.Description,
					Transport:     s.Transport,
					Command:       s.Command,
					Args:          s.Args,
					Env:           s.Env,
					URL:           s.URL,
					AuthSecretRef: s.AuthSecretRef,
				}
				_, err := h.keeperClient.MCPServers().Update(ctx, s.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[s.ID] = true
			} else {
				createReq := outbound.CreateMCPServerRequest{
					Name:          s.Name,
					Description:   s.Description,
					Transport:     s.Transport,
					Command:       s.Command,
					Args:          s.Args,
					Env:           s.Env,
					URL:           s.URL,
					AuthSecretRef: s.AuthSecretRef,
				}
				newServer, err := h.keeperClient.MCPServers().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newServer.ID] = true
			}
		}
		for _, s := range currentServers {
			if !processedIDs[s.ID] {
				err := h.keeperClient.MCPServers().Delete(ctx, s.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	} else if isSkill {
		var reqSkills []*outbound.Skill
		if err := json.Unmarshal(dataBytes, &reqSkills); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		currentSkills, err := h.keeperClient.Skills().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		currentMap := make(map[uuid.UUID]*outbound.Skill)
		for _, s := range currentSkills {
			currentMap[s.ID] = s
		}
		processedIDs := make(map[uuid.UUID]bool)
		for _, s := range reqSkills {
			if s.ID != uuid.Nil && currentMap[s.ID] != nil {
				updateReq := outbound.UpdateSkillRequest{
					Name:        s.Name,
					Description: s.Description,
					Content:     s.Content,
				}
				_, err := h.keeperClient.Skills().Update(ctx, s.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[s.ID] = true
			} else {
				createReq := outbound.CreateSkillRequest{
					Name:        s.Name,
					Description: s.Description,
					Content:     s.Content,
				}
				newSkill, err := h.keeperClient.Skills().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newSkill.ID] = true
			}
		}
		for _, s := range currentSkills {
			if !processedIDs[s.ID] {
				err := h.keeperClient.Skills().Delete(ctx, s.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	} else if isPrompt {
		var reqPrompts []*outbound.PromptTemplate
		if err := json.Unmarshal(dataBytes, &reqPrompts); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		currentPrompts, err := h.keeperClient.Prompts().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		currentMap := make(map[uuid.UUID]*outbound.PromptTemplate)
		for _, p := range currentPrompts {
			currentMap[p.ID] = p
		}
		processedIDs := make(map[uuid.UUID]bool)
		for _, p := range reqPrompts {
			if p.ID != uuid.Nil && currentMap[p.ID] != nil {
				updateReq := outbound.UpdatePromptTemplateRequest{
					Name:    p.Name,
					Content: p.Content,
					Status:  p.Status,
				}
				_, err := h.keeperClient.Prompts().Update(ctx, p.ID, &updateReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[p.ID] = true
			} else {
				createReq := outbound.CreatePromptTemplateRequest{
					Name:    p.Name,
					Content: p.Content,
					Status:  p.Status,
				}
				newPrompt, err := h.keeperClient.Prompts().Create(ctx, &createReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				processedIDs[newPrompt.ID] = true
			}
		}
		for _, p := range currentPrompts {
			if !processedIDs[p.ID] {
				err := h.keeperClient.Prompts().Delete(ctx, p.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
	} else {
		type CommunityWithAgents struct {
			outbound.Community
			Agents []string `json:"agents"`
		}
		var reqComms []*CommunityWithAgents
		if err := json.Unmarshal(dataBytes, &reqComms); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		currentComms, err := h.keeperClient.Communities().List(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// List current agents to compare and update community assignments
		currentAgents, err := h.keeperClient.Agents().List(ctx)
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
			var commID uuid.UUID
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
				commID = comm.ID
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
				commID = newComm.ID
			}

			// Synchronize agent community assignments
			targetAgents := make(map[uuid.UUID]bool)
			for _, aStr := range comm.Agents {
				aID, err := uuid.Parse(aStr)
				if err == nil {
					targetAgents[aID] = true
				}
			}

			for _, agent := range currentAgents {
				isTarget := targetAgents[agent.ID]
				isCurrentlyAssigned := agent.CommunityID != nil && *agent.CommunityID == commID

				if isTarget && !isCurrentlyAssigned {
					_ = h.keeperClient.Communities().AssignAgent(ctx, commID, agent.ID)
				} else if !isTarget && isCurrentlyAssigned {
					_ = h.keeperClient.Communities().UnassignAgent(ctx, commID, agent.ID)
				}
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

// ListLLMBindings lists all LLM bindings
func (h *ConfiguratorHandler) ListLLMBindings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	bindings, err := h.keeperClient.LLMBindings().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindings)
}

// CreateLLMBinding creates a new LLM binding
func (h *ConfiguratorHandler) CreateLLMBinding(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req outbound.CreateLLMBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binding, err := h.keeperClient.LLMBindings().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, binding)
}

// ListPrompts lists all prompt templates
func (h *ConfiguratorHandler) ListPrompts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	prompts, err := h.keeperClient.Prompts().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prompts)
}

// CreatePrompt creates a new prompt template
func (h *ConfiguratorHandler) CreatePrompt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req outbound.CreatePromptTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt, err := h.keeperClient.Prompts().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, prompt)
}

// ListSkills lists all skill definitions
func (h *ConfiguratorHandler) ListSkills(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	skills, err := h.keeperClient.Skills().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, skills)
}

// CreateSkill creates a new skill definition
func (h *ConfiguratorHandler) CreateSkill(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req outbound.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	skill, err := h.keeperClient.Skills().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, skill)
}

// ListMCPServers lists all MCP servers
func (h *ConfiguratorHandler) ListMCPServers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	servers, err := h.keeperClient.MCPServers().List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

// CreateMCPServer creates a new MCP server
func (h *ConfiguratorHandler) CreateMCPServer(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req outbound.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.keeperClient.MCPServers().Create(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, server)
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
