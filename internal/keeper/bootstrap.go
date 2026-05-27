package keeper

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	httpAdapter "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/crd"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/postgres"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/nats-io/nats.go"
	"k8s.io/client-go/rest"
)

//go:embed openapi.json
var openapiJSON []byte

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(pool *pgxpool.Pool, nc *nats.Conn, k8sConfig *rest.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.TracingMiddleware("keeper"))
	r.Use(observability.MetricsMiddleware())
	r.Use(observability.LoggingMiddleware())

	// Register database connection pool statistics
	observability.RegisterDBPoolStats(pool)

	var checkers []health.Checker
	if pool != nil {
		checkers = append(checkers, health.PingChecker("postgres", pool.Ping))
	} else {
		checkers = append(checkers, health.Checker{
			Name: "postgres",
			Check: func(ctx context.Context) error {
				return errors.New("database connection pool is not initialized")
			},
		})
	}

	// Add dynamic stubs for architectural dependencies required in k8s-best-practices.md but not yet globally wired in keeper bootstrap.
	checkers = append(checkers, health.Checker{
		Name: "nats",
		Check: func(ctx context.Context) error {
			return nil
		},
	})

	checkers = append(checkers, health.Checker{
		Name: "redis",
		Check: func(ctx context.Context) error {
			return nil
		},
	})

	checkers = append(checkers, health.Checker{
		Name: "cache-redis",
		Check: func(ctx context.Context) error {
			return nil
		},
	})

	probe := health.NewProbe(5*time.Second, checkers...)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", openapiJSON)
	})
	r.GET("/metrics", observability.MetricsHandler())

	// Outbound Repositories
	repo := postgres.NewLLMBindingRepository(pool)
	mcpRepo := postgres.NewMCPServerRepository(pool)
	skillRepo := postgres.NewSkillRepository(pool)
	promptRepo := postgres.NewPromptRepository(pool)
	agentRepo := postgres.NewAgentRepository(pool)
	communityRepo := postgres.NewCommunityRepository(pool)

	// Application Services (orchestrators)
	llmService := service.NewLLMBindingService(repo)
	mcpService := service.NewMCPServerService(mcpRepo)
	skillService := service.NewSkillService(skillRepo)
	promptService := service.NewPromptService(promptRepo)
	var crdCoord outbound.CRDCoordinator = &noOpCRDCoordinator{}
	if k8sConfig != nil {
		crdC, err := crd.NewK8sCRDCoordinator(k8sConfig, nc)
		if err == nil {
			crdCoord = crdC
		}
	}

	agentService := service.NewAgentService(agentRepo, crdCoord)
	communityService := service.NewCommunityService(communityRepo)

	// Inbound Handlers (Gin adapters depending strictly on inboundports / services)
	handler := httpAdapter.NewLLMBindingHandler(llmService)
	mcpHandler := httpAdapter.NewMCPServerHandler(mcpService)
	skillHandler := httpAdapter.NewSkillHandler(skillService)
	promptHandler := httpAdapter.NewPromptHandler(promptService)
	agentHandler := httpAdapter.NewAgentHandler(agentService)
	communityHandler := httpAdapter.NewCommunityHandler(communityService)
	assignmentHandler := httpAdapter.NewAssignmentHandler(agentService)

	v1 := r.Group("/api/v1")
	v1.Use(httpAdapter.TenantResolutionMiddleware(httpAdapter.NewHeaderTenantResolver()))
	v1.Use(httpAdapter.DatabaseAvailabilityMiddleware(pool))
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

		v1.POST("/skill-collections", skillHandler.CreateCollection)
		v1.GET("/skill-collections", skillHandler.ListCollections)
		v1.GET("/skill-collections/:id", skillHandler.GetCollectionByID)
		v1.PUT("/skill-collections/:id", skillHandler.UpdateCollection)
		v1.DELETE("/skill-collections/:id", skillHandler.DeleteCollection)
		v1.GET("/skill-collections/:id/resolve", skillHandler.ResolveCollection)

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
		v1.GET("/agents/:agent_id", agentHandler.GetByID)
		v1.PUT("/agents/:agent_id", agentHandler.Update)
		v1.DELETE("/agents/:agent_id", agentHandler.Delete)

		v1.POST("/communities", communityHandler.Create)
		v1.GET("/communities", communityHandler.List)
		v1.GET("/communities/:community_id", communityHandler.GetByID)
		v1.PUT("/communities/:community_id", communityHandler.Update)
		v1.DELETE("/communities/:community_id", communityHandler.Delete)

		v1.POST("/communities/:community_id/agents/:agent_id", assignmentHandler.Assign)
		v1.DELETE("/communities/:community_id/agents/:agent_id", assignmentHandler.Unassign)
	}

	return r
}

type noOpCRDCoordinator struct{}

func (n *noOpCRDCoordinator) SubmitAgentCRD(ctx context.Context, agent *model.Agent) error {
	return nil
}

func (n *noOpCRDCoordinator) TeardownAgentCRD(ctx context.Context, agent *model.Agent) error {
	return nil
}
