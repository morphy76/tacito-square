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
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/shutdown"
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

	echoSubscriber := agent.NewEchoSubscriber(nc, agentName, communityRef, "", logger)
	if err := echoSubscriber.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start echo subscriber")
	}
	mgr.Register("echo-subscriber", func(ctx context.Context) error {
		logger.Info().Msg("stopping echo subscriber")
		return echoSubscriber.Stop()
	})

	// 6. Create HTTP router
	router := agent.NewServer()

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
