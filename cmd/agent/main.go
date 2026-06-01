package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/morphy76/tacito-square/internal/agent"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/ollama"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/openai"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/qdrant"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/redis"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/application/service"
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/shutdown"
	natsclient "github.com/nats-io/nats.go"
)

// Version is the application version, set at build time or using fallback.
var Version = "0.1.0"

func main() {
	// 1. Load configuration
	v, err := config.Load("TS_AGENT")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	v.SetDefault("port", "8081")
	v.SetDefault("qdrant.collection.name", "ts_agent_memories")
	v.SetDefault("qdrant.vector.dimension", 1536)
	v.SetDefault("max.reasoning.steps", 5)
	v.SetDefault("stm.limit", 10)
	v.SetDefault("system.prompt", "")

	port := v.GetString("port")
	logLevel := v.GetString("log.level")
	otelEndpoint := v.GetString("otel.endpoint")

	// 2. Initialize structured logging
	logger := observability.NewLogger(logLevel, os.Stdout)

	logger.Info().
		Str("component", "agent").
		Str("version", Version).
		Str("port", port).
		Msg("starting component")

	// 3. Initialize OpenTelemetry tracing
	ctx := context.Background()
	shutdownTracer, err := observability.InitTracer(ctx, "agent", Version, otelEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize tracer")
	}

	// 4. Initialize shutdown manager (30 seconds graceful shutdown timeout)
	mgr := shutdown.NewManager(30 * time.Second)

	// Register tracer shutdown (registered first, runs last)
	mgr.Register("otel-tracer", func(ctx context.Context) error {
		return shutdownTracer(ctx)
	})

	// 5. Initialize NATS connection and echo subscriber
	natsURL := v.GetString("nats.url")
	if natsURL == "" {
		logger.Fatal().Msg("TS_AGENT_NATS_URL is required but not set")
	}

	agentName := v.GetString("name")
	if agentName == "" {
		logger.Fatal().Msg("TS_AGENT_NAME is required but not set")
	}

	communityRef := v.GetString("community.ref")
	if communityRef == "" {
		logger.Fatal().Msg("TS_AGENT_COMMUNITY_REF is required but not set")
	}

	nc, err := agent.ConnectNATS(natsURL, logger)
	if err != nil {
		logger.Fatal().Err(err).Str("nats.url", natsURL).Msg("failed to connect to NATS")
	}
	mgr.Register("nats-client", func(ctx context.Context) error {
		logger.Info().Msg("closing NATS connection")
		nc.Close()
		return nil
	})

	// 5. Initialize brain reasoning engine (stateless outbound port)
	provider := v.GetString("brain.provider")
	if provider == "" {
		provider = "openai"
	}

	failureThreshold := v.GetInt("brain.circuit.failure.threshold")
	recoveryTimeoutSecs := v.GetInt("brain.circuit.recovery.timeout.seconds")
	recoveryTimeout := time.Duration(recoveryTimeoutSecs) * time.Second
	timeoutSecs := v.GetInt("brain.timeout.seconds")
	timeout := time.Duration(timeoutSecs) * time.Second

	// Create a single shared thread-safe instrumented HTTP client for the outbound LLM calls
	sharedHTTPClient := observability.NewInstrumentedClient(timeout)

	var brain outbound.Brain

	switch provider {
	case "ollama":
		brain = ollama.NewAdapter(ollama.Config{
			Endpoint:         v.GetString("ollama.endpoint"),
			Model:            v.GetString("brain.model"),
			Temperature:      v.GetFloat64("brain.temperature"),
			MaxTokens:        v.GetInt("brain.max.tokens"),
			Timeout:          timeout,
			FailureThreshold: failureThreshold,
			RecoveryTimeout:  recoveryTimeout,
			FallbackMessage:  "ollama brain fallback response",
			HTTPClient:       sharedHTTPClient,
		})
	default:
		brain = openai.NewAdapter(openai.Config{
			Endpoint:         v.GetString("openai.endpoint"),
			APIKey:           v.GetString("openai.api.key"),
			Model:            v.GetString("brain.model"),
			Temperature:      v.GetFloat64("brain.temperature"),
			MaxTokens:        v.GetInt("brain.max.tokens"),
			Timeout:          timeout,
			FailureThreshold: failureThreshold,
			RecoveryTimeout:  recoveryTimeout,
			FallbackMessage:  "openai brain fallback response",
			HTTPClient:       sharedHTTPClient,
		})
	}

	// 5. Initialize Redis short-term memory adapter
	redisURL := v.GetString("redis.url")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	ttlSecs := v.GetInt("stm.ttl.seconds")
	if ttlSecs == 0 {
		ttlSecs = 86400 // 24 hours default
	}
	ttl := time.Duration(ttlSecs) * time.Second

	memoryAdapter, err := redis.NewRedisMemoryAdapter(redisURL, ttl)
	if err != nil {
		logger.Fatal().Err(err).Str("redis.url", redisURL).Msg("failed to connect to Redis short-term memory")
	}
	mgr.Register("redis-client", func(ctx context.Context) error {
		logger.Info().Msg("closing Redis memory connection")
		return memoryAdapter.Close()
	})

	// 5b. Initialize Qdrant long-term memory adapter
	var ltm outbound.LongTermMemory
	var embedder outbound.Embedder
	var qdrantAdapter *qdrant.QdrantLTMAdapter
	var hasQdrant bool

	qdrantURL := v.GetString("qdrant.url")
	if qdrantURL != "" {
		collectionName := v.GetString("qdrant.collection.name")
		vectorDim := v.GetInt("qdrant.vector.dimension")

		logger.Info().
			Str("qdrant.url", qdrantURL).
			Str("qdrant.collection", collectionName).
			Int("qdrant.dimension", vectorDim).
			Msg("initializing Qdrant long-term memory")

		var err error
		qdrantAdapter, err = qdrant.NewQdrantLTMAdapter(qdrantURL, collectionName, uint64(vectorDim))
		if err != nil {
			logger.Fatal().Err(err).Str("qdrant.url", qdrantURL).Msg("failed to connect to Qdrant long-term memory")
		}

		mgr.Register("qdrant-client", func(ctx context.Context) error {
			logger.Info().Msg("closing Qdrant memory connection")
			return qdrantAdapter.Close()
		})

		ltm = qdrantAdapter
		hasQdrant = true

		var ok bool
		embedder, ok = brain.(outbound.Embedder)
		if !ok {
			logger.Fatal().Msg("Brain adapter does not implement Embedder interface")
		}
	}

	// 5c. Initialize Cognitive reasoning engine loop
	maxReasoningSteps := v.GetInt("max.reasoning.steps")
	stmLimit := v.GetInt("stm.limit")
	systemPrompt := v.GetString("system.prompt")

	cogEngine := service.NewCognitiveEngine(brain, maxReasoningSteps)
	if hasQdrant {
		cogEngine = cogEngine.WithLTM(embedder, ltm)
	}
	natsPublisher := nats.NewNATSEventPublisher(nc)
	cogEngine = cogEngine.WithPublisher(natsPublisher)

	// TODO: Remove mock tools when the agent is ready for production.
	// Register Always Used mock skill/tool: utility_ping
	cogEngine.RegisterTool("utility_ping", func(ctx context.Context, args map[string]any) (string, error) {
		return "pong", nil
	})

	// Register Never Used mock skill collection: restricted
	cogEngine.RegisterSkill(service.Skill{
		Name:        "restricted",
		Description: "Restricted capabilities that require special authorization.",
		Content:     "Warning: Restricted access instructions. Under no circumstances should you perform unauthorized queries.",
	})

	// Register Sometimes Used mock skill collection: math
	cogEngine.RegisterSkill(service.Skill{
		Name:        "math",
		Description: "Dynamic math instructions for verifying calculations.",
		Content:     "Math Guidelines: Always verify additions step-by-step and format them clearly.",
	})

	processor := service.NewMessageProcessorService(brain, memoryAdapter, ltm, embedder, cogEngine, stmLimit, systemPrompt)

	echoSubscriber := agent.NewEchoSubscriber(nc, agentName, communityRef, "", processor, logger)
	if err := echoSubscriber.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start echo subscriber")
	}
	mgr.Register("echo-subscriber", func(ctx context.Context) error {
		logger.Info().Msg("stopping echo subscriber")
		return echoSubscriber.Stop()
	})

	// 6. Create HTTP router with parallel readiness dependency checkers
	var checkers []health.Checker

	// NATS Checker
	checkers = append(checkers, health.Checker{
		Name: "nats",
		Check: func(ctx context.Context) error {
			if nc == nil || nc.Status() != natsclient.CONNECTED {
				return errors.New("NATS connection is offline")
			}
			return nil
		},
	})

	// Redis Checker
	checkers = append(checkers, health.Checker{
		Name:  "redis",
		Check: memoryAdapter.Ping,
	})

	// Cache Redis Checker (stub)
	checkers = append(checkers, health.Checker{
		Name: "cache-redis",
		Check: func(ctx context.Context) error {
			return nil
		},
	})

	// Qdrant Checker
	if hasQdrant && qdrantAdapter != nil {
		checkers = append(checkers, health.Checker{
			Name:  "qdrant",
			Check: qdrantAdapter.Ping,
		})
	}

	router := agent.NewServer(checkers...)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP server in the background
	go func() {
		logger.Info().Str("addr", srv.Addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Register HTTP server shutdown (registered last, runs first)
	mgr.Register("http-server", func(ctx context.Context) error {
		logger.Info().Msg("shutting down HTTP server")
		return srv.Shutdown(ctx)
	})

	// 7. Block until termination signal and execute cleanup
	logger.Info().Msg("component is ready")
	if err := mgr.Wait(syscall.SIGINT, syscall.SIGTERM); err != nil {
		logger.Error().Err(err).Msg("error during graceful shutdown")
		os.Exit(1)
	}

	logger.Info().Msg("component stopped gracefully")
}
