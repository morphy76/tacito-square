package keeper

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	httpAdapter "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/http"
	keepernats "github.com/morphy76/tacito-square/internal/keeper/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/crd"
	outboundNats "github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/outbound/postgres"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/application/service"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/adapters/outbound/cache"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	sharedports "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/nats-io/nats.go"
	"github.com/morphy76/tacito-square/api/openapi"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"k8s.io/client-go/rest"
)

var openapiJSON = openapi.Spec

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(
	pool *pgxpool.Pool,
	nc *nats.Conn,
	redisClient redis.UniversalClient,
	k8sConfig *rest.Config,
	logger zerolog.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.TracingMiddleware("keeper"))
	r.Use(observability.MetricsMiddleware())
	r.Use(observability.LoggingMiddleware())

	// Register database connection pool statistics
	observability.RegisterDBPoolStats(pool)

	var cacheClient sharedports.Cache
	if redisClient != nil {
		redisWrapper := cache.NewRedisClientWrapper(redisClient)
		cacheClient = cache.NewCacheAdapter(redisWrapper, "keeper")
	} else {
		// Use InMemoryRedis fallback cache
		inMemRedis := cache.NewInMemoryRedis()
		cacheClient = cache.NewCacheAdapter(inMemRedis, "keeper")
	}

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

	checkers = append(checkers, health.Checker{
		Name: "nats",
		Check: func(ctx context.Context) error {
			if nc == nil || nc.Status() != nats.CONNECTED {
				return errors.New("NATS connection is offline")
			}
			return nil
		},
	})

	checkers = append(checkers, health.Checker{
		Name: "redis",
		Check: func(ctx context.Context) error {
			if redisClient == nil {
				return errors.New("redis client is not initialized")
			}
			return redisClient.Ping(ctx).Err()
		},
	})

	checkers = append(checkers, health.Checker{
		Name: "cache-redis",
		Check: func(ctx context.Context) error {
			if redisClient == nil {
				return errors.New("cache redis client is not initialized")
			}
			return redisClient.Ping(ctx).Err()
		},
	})

	probe := health.NewProbe(5*time.Second, checkers...)

	// Precompute ETag for OpenAPI JSON
	hash := sha256.Sum256(openapiJSON)
	etag := fmt.Sprintf(`"%x"`, hash)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
		c.Header("ETag", etag)
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", openapiJSON)
	})
	r.GET("/metrics", observability.MetricsHandler())

	// Outbound Repositories
	repo := postgres.NewLLMBindingRepository(pool)
	mcpRepo := postgres.NewMCPClientRepository(pool)
	skillRepo := postgres.NewSkillRepository(pool)
	promptRepo := postgres.NewPromptRepository(pool)
	agentRepo := postgres.NewAgentRepository(pool)
	communityRepo := postgres.NewCommunityRepository(pool)

	// Application Services (orchestrators)
	llmService := service.NewLLMBindingService(repo, cacheClient)
	mcpService := service.NewMCPClientService(mcpRepo)
	skillService := service.NewSkillService(skillRepo)
	promptService := service.NewPromptService(promptRepo)
	var crdCoord outbound.CRDCoordinator = &noOpCRDCoordinator{}
	if k8sConfig != nil {
		crdC, err := crd.NewK8sCRDCoordinator(k8sConfig, repo, promptRepo, skillRepo, mcpRepo, nc)
		if err == nil {
			crdCoord = crdC
		}
	}

	// Events feature
	eventPublisher := outboundNats.NewNATSEventPublisher(nc)
	eventSubscriber := outboundNats.NewNATSEventSubscriber(nc)
	eventService := service.NewEventService(eventPublisher, eventSubscriber, communityRepo)

	agentService := service.NewAgentService(agentRepo, communityRepo, crdCoord, cacheClient, eventPublisher, repo, promptRepo)
	communityService := service.NewCommunityService(communityRepo, nil)
	lifecycleService := service.NewLifecycleService(agentRepo, communityRepo, crdCoord, nc)

	// Wire NATS registry subscriber, registry pruner, and registry handler
	if nc != nil && pool != nil {
		subscriber := keepernats.NewRegistrySubscriber(nc, agentRepo, cacheClient, eventPublisher, logger)
		pruner := service.NewRegistryPruner(agentRepo, cacheClient, eventPublisher, logger)
		handler := keepernats.NewRegistryHandler(nc, agentRepo, cacheClient, logger)

		ctx := context.Background()
		if err := subscriber.Start(ctx); err != nil {
			logger.Error().Err(err).Msg("failed to start registry subscriber")
		}
		if err := pruner.Start(ctx); err != nil {
			logger.Error().Err(err).Msg("failed to start registry pruner")
		}
		if err := handler.Start(ctx); err != nil {
			logger.Error().Err(err).Msg("failed to start registry handler")
		}

		nc.SetClosedHandler(func(conn *nats.Conn) {
			logger.Info().Msg("NATS connection closed, stopping registry subscriber, pruner, and handler")
			if err := subscriber.Stop(); err != nil {
				logger.Error().Err(err).Msg("error stopping registry subscriber")
			}
			if err := pruner.Stop(); err != nil {
				logger.Error().Err(err).Msg("error stopping registry pruner")
			}
			if err := handler.Stop(); err != nil {
				logger.Error().Err(err).Msg("error stopping registry handler")
			}
		})
	}

	// Inbound Handlers (Gin adapters depending strictly on inboundports / services)
	handler := httpAdapter.NewLLMBindingHandler(llmService)
	mcpHandler := httpAdapter.NewMCPClientHandler(mcpService)
	skillHandler := httpAdapter.NewSkillHandler(skillService)
	promptHandler := httpAdapter.NewPromptHandler(promptService)
	agentHandler := httpAdapter.NewAgentHandler(agentService)
	communityHandler := httpAdapter.NewCommunityHandler(communityService)
	assignmentHandler := httpAdapter.NewAssignmentHandler(communityService)
	lifecycleHandler := httpAdapter.NewLifecycleHandler(lifecycleService)
	eventHandler := httpAdapter.NewEventHandler(eventService, eventService)
	cardHandler := httpAdapter.NewCardHandler(agentRepo, communityRepo)

	v1 := r.Group("/api/v1")
	v1.Use(httpAdapter.TenantResolutionMiddleware(httpAdapter.NewHeaderTenantResolver()))
	v1.Use(httpAdapter.DatabaseAvailabilityMiddleware(pool))
	{
		v1.POST("/llm-bindings", handler.Create)
		v1.GET("/llm-bindings", handler.List)
		v1.GET("/llm-bindings/:id", handler.GetByID)
		v1.PUT("/llm-bindings/:id", handler.Update)
		v1.DELETE("/llm-bindings/:id", handler.Delete)

		v1.POST("/mcp-clients", mcpHandler.Create)
		v1.GET("/mcp-clients", mcpHandler.List)
		v1.GET("/mcp-clients/:id", mcpHandler.GetByID)
		v1.PUT("/mcp-clients/:id", mcpHandler.Update)
		v1.DELETE("/mcp-clients/:id", mcpHandler.Delete)

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
		v1.POST("/prompt-collections/:id/prompts/:prompt_id", promptHandler.AddPromptToCollection)
		v1.DELETE("/prompt-collections/:id/prompts/:prompt_id", promptHandler.RemovePromptFromCollection)

		v1.POST("/agents", agentHandler.Create)
		v1.GET("/agents", agentHandler.List)
		v1.GET("/agents/:agent_id", agentHandler.GetByID)
		v1.PUT("/agents/:agent_id", agentHandler.Update)
		v1.DELETE("/agents/:agent_id", agentHandler.Delete)
		v1.POST("/agents/:agent_id/prompts/:prompt_id", agentHandler.AttachPromptToAgent)
		v1.DELETE("/agents/:agent_id/prompts/:prompt_id", agentHandler.DetachPromptFromAgent)
		v1.POST("/agents/:agent_id/prompt-collections/:collection_id", agentHandler.AttachCollectionToAgent)
		v1.DELETE("/agents/:agent_id/prompt-collections/:collection_id", agentHandler.DetachCollectionFromAgent)
		v1.GET("/agents/:agent_id/prompts", agentHandler.ResolveEffectivePrompts)

		v1.POST("/communities", communityHandler.Create)
		v1.GET("/communities", communityHandler.List)
		v1.GET("/communities/:community_id", communityHandler.GetByID)
		v1.PUT("/communities/:community_id", communityHandler.Update)
		v1.DELETE("/communities/:community_id", communityHandler.Delete)

		v1.POST("/communities/:community_id/agents", assignmentHandler.Assign)
		v1.POST("/communities/:community_id/agents/:agent_id", assignmentHandler.Assign)
		v1.GET("/communities/:community_id/agents", assignmentHandler.ListAssignments)
		v1.DELETE("/communities/:community_id/agents/:agent_id", assignmentHandler.Unassign)

		// Agent & Community Lifecycle Management routes
		v1.POST("/agents/:agent_id/deploy", lifecycleHandler.DeployAgent)
		v1.POST("/agents/:agent_id/undeploy", lifecycleHandler.UndeployAgent)
		v1.GET("/agents/:agent_id/status", lifecycleHandler.GetAgentStatus)

		v1.POST("/communities/:community_id/deploy", lifecycleHandler.DeployCommunity)
		v1.POST("/communities/:community_id/undeploy", lifecycleHandler.UndeployCommunity)
		v1.GET("/communities/:community_id/status", lifecycleHandler.GetCommunityStatus)

		// Event routes
		v1.POST("/events", eventHandler.PublishEvent)
		v1.GET("/events/stream", eventHandler.StreamEvents)

		// Discovery well-known routes
		v1.GET("/communities/:community_id/agents/:agent_id/.well-known/agent-card.json", cardHandler.GetAgentCard)
		v1.GET("/communities/:community_id/.well-known/community-card.json", cardHandler.GetCommunityCard)
		v1.GET("/communities/:community_id/.well-known/agent-cards.json", cardHandler.GetAgentCards)
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

func (n *noOpCRDCoordinator) GetAgentCRDStatus(ctx context.Context, agentID uuid.UUID) (*v1alpha1.TacitoAgentStatus, error) {
	return nil, nil
}

