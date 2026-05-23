package keeper

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	httpAdapter "github.com/morphy76/tacito-square/internal/keeper/adapters/http"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/postgres"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
)

//go:embed openapi.json
var openapiJSON []byte

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(pool *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.MetricsMiddleware())

	var checkers []health.Checker
	if pool != nil {
		checkers = append(checkers, health.PingChecker("postgres", pool.Ping))
	}

	probe := health.NewProbe(5*time.Second, checkers...)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", openapiJSON)
	})
	r.GET("/metrics", observability.MetricsHandler())

	if pool != nil {
		repo := postgres.NewLLMBindingRepository(pool)
		handler := httpAdapter.NewLLMBindingHandler(repo)

		mcpRepo := postgres.NewMCPServerRepository(pool)
		mcpHandler := httpAdapter.NewMCPServerHandler(mcpRepo)

		skillRepo := postgres.NewSkillRepository(pool)
		skillHandler := httpAdapter.NewSkillHandler(skillRepo)

		promptRepo := postgres.NewPromptRepository(pool)
		promptHandler := httpAdapter.NewPromptHandler(promptRepo)

		agentRepo := postgres.NewAgentRepository(pool)
		agentHandler := httpAdapter.NewAgentHandler(agentRepo)

		v1 := r.Group("/api/v1")
		v1.Use(httpAdapter.TenantResolutionMiddleware(httpAdapter.NewHeaderTenantResolver()))
		{
			v1.POST("/llm-bindings", handler.Create)
			v1.GET("/llm-bindings", handler.List)
			v1.GET("/llm-bindings/:id", handler.GetByID)
			v1.PUT("/llm-bindings/:id", handler.Update)
			v1.DELETE("/llm-bindings/:id", handler.Delete)

			v1.POST("/mcp-servers", mcpHandler.Create)
			v1.GET("/mcp-servers", mcpHandler.List)
			v1.GET("/mcp-servers/:id", mcpHandler.GetByID)
			v1.PUT("/mcp-servers/:id", mcpHandler.Update)
			v1.DELETE("/mcp-servers/:id", mcpHandler.Delete)

			v1.POST("/skills", skillHandler.Create)
			v1.GET("/skills", skillHandler.List)
			v1.GET("/skills/:id", skillHandler.GetByID)
			v1.PUT("/skills/:id", skillHandler.Update)
			v1.DELETE("/skills/:id", skillHandler.Delete)

			v1.POST("/agents/:agent_id/skills/:skill_id", skillHandler.AttachSkillToAgent)
			v1.DELETE("/agents/:agent_id/skills/:skill_id", skillHandler.DetachSkillFromAgent)

			v1.POST("/prompts", promptHandler.CreateTemplate)
			v1.GET("/prompts", promptHandler.ListTemplates)
			v1.GET("/prompts/:id", promptHandler.GetTemplateByID)
			v1.PUT("/prompts/:id", promptHandler.UpdateTemplate)
			v1.DELETE("/prompts/:id", promptHandler.DeleteTemplate)

			v1.POST("/prompt-collections", promptHandler.CreateCollection)
			v1.GET("/prompt-collections", promptHandler.ListCollections)
			v1.GET("/prompt-collections/:id", promptHandler.GetCollectionByID)
			v1.PUT("/prompt-collections/:id", promptHandler.UpdateCollection)
			v1.DELETE("/prompt-collections/:id", promptHandler.DeleteCollection)
			v1.GET("/prompt-collections/:id/resolve", promptHandler.ResolveCollection)

			v1.POST("/agents", agentHandler.Create)
			v1.GET("/agents", agentHandler.List)
			v1.GET("/agents/:id", agentHandler.GetByID)
			v1.PUT("/agents/:id", agentHandler.Update)
			v1.DELETE("/agents/:id", agentHandler.Delete)
		}
	}

	return r
}
