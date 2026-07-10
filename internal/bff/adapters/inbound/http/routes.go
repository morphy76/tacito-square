package http

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

func RegisterRoutes(
	r *gin.Engine,
	sessionUC inbound.SessionUseCase,
	eventUC inbound.EventStreamUseCase,
	keeperClient outbound.KeeperClient,
	uiPath string,
) {
	auth := NewAuthHandler(sessionUC, uiPath)
	sse := NewSSEHandler(eventUC)
	configuratorHandler := NewConfiguratorHandler(keeperClient)

	apiPrefix := "/api/v1"
	if uiPath != "" && uiPath != "/" {
		apiPrefix = strings.TrimSuffix(uiPath, "/") + "/api/v1"
	}
	v1 := r.Group(apiPrefix)
	{
		// Public auth endpoints
		v1.GET("/auth/login", auth.Login)
		v1.GET("/auth/callback", auth.Callback)
		v1.POST("/auth/backchannel-logout", auth.BackchannelLogout)

		// Protected group
		protected := v1.Group("")
		protected.Use(SessionMiddleware(sessionUC, uiPath))
		{
			protected.POST("/auth/logout", auth.Logout)
			protected.GET("/auth/me", auth.Me)
			protected.GET("/events/stream", sse.StreamEvents)
		}

		// Configurator group (protected by session and role checking)
		configurator := v1.Group("/configurator")
		configurator.Use(SessionMiddleware(sessionUC, uiPath))
		configurator.Use(RequireRoles("keeper-admin", "agent-spawner"))
		{
			configurator.GET("/wizard/options", configuratorHandler.GetWizardOptions)

			// LLM Bindings CRUD/Listing
			configurator.GET("/llm-bindings", configuratorHandler.ListLLMBindings)
			configurator.POST("/llm-bindings", configuratorHandler.CreateLLMBinding)

			// Prompts CRUD/Listing
			configurator.GET("/prompts", configuratorHandler.ListPrompts)
			configurator.POST("/prompts", configuratorHandler.CreatePrompt)

			// Skills CRUD/Listing
			configurator.GET("/skills", configuratorHandler.ListSkills)
			configurator.POST("/skills", configuratorHandler.CreateSkill)

			// MCP Servers CRUD/Listing
			configurator.GET("/mcp-servers", configuratorHandler.ListMCPServers)
			configurator.POST("/mcp-servers", configuratorHandler.CreateMCPServer)

			// Agents CRUD
			configurator.GET("/agents", configuratorHandler.ListAgents)
			configurator.GET("/agents/:id", configuratorHandler.GetAgent)
			configurator.POST("/agents", configuratorHandler.CreateAgent)
			configurator.PUT("/agents/:id", configuratorHandler.UpdateAgent)
			configurator.DELETE("/agents/:id", configuratorHandler.DeleteAgent)

			// Communities CRUD
			configurator.GET("/communities", configuratorHandler.ListCommunities)
			configurator.GET("/communities/:id", configuratorHandler.GetCommunity)
			configurator.POST("/communities", configuratorHandler.CreateCommunity)
			configurator.PUT("/communities/:id", configuratorHandler.UpdateCommunity)
			configurator.DELETE("/communities/:id", configuratorHandler.DeleteCommunity)

			// Assignments
			configurator.POST("/communities/:id/agents/:agent_id", configuratorHandler.AssignAgent)
			configurator.DELETE("/communities/:id/agents/:agent_id", configuratorHandler.UnassignAgent)

			// Advanced Sync
			configurator.POST("/advanced-sync", configuratorHandler.AdvancedSync)
		}

		auditor := v1.Group("/auditor")
		auditor.Use(SessionMiddleware(sessionUC, uiPath))
		{
			// Future auditor routes
		}
	}
}
