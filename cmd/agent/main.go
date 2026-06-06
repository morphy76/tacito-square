package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/morphy76/tacito-square/internal/agent"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/mcp"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/nats"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/ollama"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/openai"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/qdrant"
	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/redis"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/agent/application/service"
	s3adapter "github.com/morphy76/tacito-square/internal/shared/adapters/outbound/s3"
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	sharedoutbound "github.com/morphy76/tacito-square/internal/shared/ports/outbound"
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
	v.SetDefault("mcp.timeout.seconds", 10)
	v.SetDefault("mcp.cb.failure.threshold", 5)
	v.SetDefault("mcp.cb.recovery.timeout.seconds", 15)
	v.SetDefault("mcp.clients", "")
	v.SetDefault("s3.enabled", false)
	v.SetDefault("s3.endpoint", "http://localhost:9000")
	v.SetDefault("s3.bucket", "tacito")
	v.SetDefault("s3.max.read.size", 5*1024*1024)
	v.SetDefault("s3.chunk.size", 32*1024)
	v.SetDefault("bypass.ltm", true)

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

	agentID := v.GetString("id")
	if agentID == "" {
		logger.Fatal().Msg("TS_AGENT_ID is required but not set")
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
			AgentName:        agentName,
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
			AgentName:        agentName,
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

	// 5bb. Initialize MCP adapter if clients are configured
	mcpClientsJSON := v.GetString("mcp.clients")
	var mcpClients []outbound.MCPClientInfo
	if mcpClientsJSON != "" {
		if err := json.Unmarshal([]byte(mcpClientsJSON), &mcpClients); err != nil {
			logger.Error().Err(err).Msg("failed to parse TS_AGENT_MCP_CLIENTS JSON")
		}
	}

	var mcpAdapter *mcp.MCPAdapter
	if len(mcpClients) > 0 {
		mcpTimeoutSecs := v.GetInt("mcp.timeout.seconds")
		mcpTimeout := time.Duration(mcpTimeoutSecs) * time.Second

		mcpCBThreshold := v.GetInt("mcp.cb.failure.threshold")
		mcpCBRecoverySecs := v.GetInt("mcp.cb.recovery.timeout.seconds")
		mcpCBRecovery := time.Duration(mcpCBRecoverySecs) * time.Second

		mcpAdapter = mcp.NewMCPAdapter(mcpClients, mcpTimeout).
			WithCircuitBreakerParams(mcpCBThreshold, mcpCBRecovery)

		mgr.Register("mcp-client-executor", func(ctx context.Context) error {
			logger.Info().Msg("closing MCP adapter connections")
			return mcpAdapter.Close(ctx)
		})
	}

	// 5c. Initialize Cognitive reasoning engine loop
	maxReasoningSteps := v.GetInt("max.reasoning.steps")
	stmLimit := v.GetInt("stm.limit")
	systemPrompt := v.GetString("system.prompt")

	cogEngine := service.NewCognitiveEngine(brain, maxReasoningSteps, v).
		WithCommunityID(communityRef)

	if mcpAdapter != nil {
		cogEngine = cogEngine.WithToolExecutor(mcpAdapter)
	}
	if hasQdrant {
		cogEngine = cogEngine.WithLTM(embedder, ltm)
	}
	natsPublisher := nats.NewNATSEventPublisher(nc)
	cogEngine = cogEngine.WithPublisher(natsPublisher)

	// Initialize S3 adapter if enabled
	s3Enabled := v.GetBool("s3.enabled")
	var blobStoreAdapter *s3adapter.BlobStoreAdapter
	var s3Client *s3adapter.S3HTTPClient

	if s3Enabled {
		s3Endpoint := v.GetString("s3.endpoint")
		s3Bucket := v.GetString("s3.bucket")
		maxReadSize := v.GetInt("s3.max.read.size")
		chunkSize := v.GetInt("s3.chunk.size")

		logger.Info().
			Str("s3.endpoint", s3Endpoint).
			Str("s3.bucket", s3Bucket).
			Msg("initializing S3/MinIO blob store adapter")

		// Create S3 HTTP Client (using instrumented HTTP client to inject Otel tracing and timeout)
		s3Client = s3adapter.NewS3HTTPClient(s3Endpoint, sharedHTTPClient)

		// Create S3 BlobStore adapter
		blobStoreAdapter = s3adapter.NewBlobStoreAdapter(s3Client, s3Bucket, s3Endpoint)

		// Configure BlobStore on cognitive engine
		cogEngine = cogEngine.WithBlobStore(blobStoreAdapter, maxReadSize, chunkSize)
	}

	processor := service.NewMessageProcessorService(brain, memoryAdapter, ltm, embedder, cogEngine, stmLimit, systemPrompt, v)

	var s3BlobStore sharedoutbound.BlobStore
	if s3Enabled {
		s3BlobStore = blobStoreAdapter
	}

	schemaRouter := service.NewSchemaRouterImpl(
		agentID,
		agentName,
		processor,
		memoryAdapter,
		ltm,
		embedder,
		brain,
		natsPublisher,
	)

	eventSubscriber := agent.NewEventSubscriber(nc, agentName, communityRef, schemaRouter, s3BlobStore, logger)
	if err := eventSubscriber.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start event subscriber")
	}
	mgr.Register("event-subscriber", func(ctx context.Context) error {
		logger.Info().Msg("stopping event subscriber")
		return eventSubscriber.Stop()
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

	// MinIO Checker
	if s3Enabled && blobStoreAdapter != nil {
		checkers = append(checkers, health.Checker{
			Name:  "minio",
			Check: blobStoreAdapter.Ping,
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
