package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/morphy76/tacito-square/internal/bff"
	"github.com/morphy76/tacito-square/internal/bff/adapters/outbound"
	"github.com/morphy76/tacito-square/internal/bff/application/service"
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/shutdown"
)

// Version is the application version, set at build time or using fallback.
var Version = "0.1.0"

func main() {
	// 1. Load configuration
	v, err := config.Load("TS_BFF")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	v.SetDefault("port", "8083")
	v.SetDefault("log.level", "info")
	v.SetDefault("redis.url", "redis://localhost:6379")
	v.SetDefault("redis.prefix", "bff")
	v.SetDefault("oidc.client_id", "tacito-bff")
	v.SetDefault("oidc.client_secret", "")
	v.SetDefault("oidc.redirect_uri", "http://localhost:8083/api/v1/auth/callback")
	v.SetDefault("oidc.issuer", "http://localhost:8080/realms/tacito")
	v.SetDefault("oidc.internal_issuer", "")
	v.SetDefault("oidc.timeout", "5s")
	v.SetDefault("oidc.circuit_breaker_max_fail", uint32(5))
	v.SetDefault("bff.ui_path", "/ui")
	v.SetDefault("bff.session.ttl", "24h")
	v.SetDefault("keeper.base_url", "http://localhost:8080")
	v.SetDefault("keeper.timeout", "5s")
	v.SetDefault("backend.base_url", "http://localhost:8080")
	v.SetDefault("backend.timeout", "5s")

	port := v.GetString("port")
	logLevel := v.GetString("log.level")
	otelEndpoint := v.GetString("otel.endpoint")

	// 2. Initialize structured logging
	logger := observability.NewLogger(logLevel, os.Stdout)

	logger.Info().
		Str("component", "bff").
		Str("version", Version).
		Str("port", port).
		Msg("starting component")

	// 3. Initialize OpenTelemetry tracing
	ctx := context.Background()
	shutdownTracer, err := observability.InitTracer(ctx, "bff", Version, otelEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize tracer")
	}

	// 4. Initialize shutdown manager (30 seconds graceful shutdown timeout)
	mgr := shutdown.NewManager(30 * time.Second)

	// Register tracer shutdown (registered first, runs last)
	mgr.Register("otel-tracer", func(ctx context.Context) error {
		return shutdownTracer(ctx)
	})

	// 5. Initialize Redis connection
	redisURL := v.GetString("redis.url")
	redisPrefix := v.GetString("redis.prefix")
	var redisClient *redis.Client
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			logger.Error().Err(err).Str("redis.url", redisURL).Msg("failed to parse Redis URL")
		} else {
			redisClient = redis.NewClient(opts)
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if pingErr := redisClient.Ping(pingCtx).Err(); pingErr != nil {
				logger.Error().Err(pingErr).Msg("failed to connect to Redis")
				redisClient.Close()
				redisClient = nil
			}
			cancel()
		}
	}

	if redisClient != nil {
		mgr.Register("redis-client", func(ctx context.Context) error {
			logger.Info().Msg("closing Redis connection")
			return redisClient.Close()
		})
	}

	// 6. Construct Outbound Adapters
	store := outbound.NewRedisSessionStore(redisClient, redisPrefix)

	oidcConfig := outbound.OIDCClientConfig{
		ClientID:              v.GetString("oidc.client_id"),
		ClientSecret:          v.GetString("oidc.client_secret"),
		RedirectURI:           v.GetString("oidc.redirect_uri"),
		Issuer:                v.GetString("oidc.issuer"),
		InternalIssuer:        v.GetString("oidc.internal_issuer"),
		Timeout:               v.GetDuration("oidc.timeout"),
		CircuitBreakerMaxFail: v.GetUint32("oidc.circuit_breaker_max_fail"),
	}
	oidcProvider := outbound.NewOIDCHTTPClient(oidcConfig)

	keeperConfig := outbound.KeeperClientConfig{
		BaseURL: v.GetString("keeper.base_url"),
		Timeout: v.GetDuration("keeper.timeout"),
	}
	keeperClient := outbound.NewKeeperHTTPClient(keeperConfig)

	backendSSEConfig := outbound.BackendSSEClientConfig{
		BaseURL: v.GetString("backend.base_url"),
		Timeout: v.GetDuration("backend.timeout"),
	}
	backendSSEClient := outbound.NewBackendSSEClient(backendSSEConfig)

	// 7. Construct Application Services
	sessionConfig := service.SessionConfig{
		ClientID:     oidcConfig.ClientID,
		ClientSecret: oidcConfig.ClientSecret,
		RedirectURI:  oidcConfig.RedirectURI,
		Issuer:       oidcConfig.Issuer,
		SessionTTL:   v.GetDuration("bff.session.ttl"),
	}
	sessionUC := service.NewSessionService(store, oidcProvider, sessionConfig)
	eventUC := service.NewEventBridgeService(backendSSEClient)

	// 8. Create HTTP server using bootstrap NewServer
	cfg := bff.Config{
		Version:      Version,
		OtelEndpoint: otelEndpoint,
		LogLevel:     logLevel,
		GinMode:      v.GetString("gin.mode"),
		UIPath:       v.GetString("bff.ui_path"),
	}
	router := bff.NewServer(cfg, sessionUC, eventUC, store, oidcProvider, keeperClient)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP server in the background
	go func() {
		logger.Info().Str("addr", srv.Addr).Msg("HTTP server listening")
		if listenErr := srv.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			logger.Error().Err(listenErr).Msg("HTTP server error")
		}
	}()

	// Register HTTP server shutdown (registered last, runs first)
	mgr.Register("http-server", func(ctx context.Context) error {
		logger.Info().Msg("shutting down HTTP server")
		return srv.Shutdown(ctx)
	})

	// 9. Block until termination signal and execute cleanup
	logger.Info().Msg("component is ready")
	if waitErr := mgr.Wait(syscall.SIGINT, syscall.SIGTERM); waitErr != nil {
		logger.Error().Err(waitErr).Msg("error during graceful shutdown")
		os.Exit(1)
	}

	logger.Info().Msg("component stopped gracefully")
}
