package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/tacito-square/internal/keeper"
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/shutdown"
)

// Version is the application version, set at build time or using fallback.
var Version = "0.1.0"

func main() {
	// 1. Load configuration
	v, err := config.Load("TS_KEEPER")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	v.SetDefault("port", "8080")

	port := v.GetString("port")
	logLevel := v.GetString("log.level")
	otelEndpoint := v.GetString("otel.endpoint")

	// 2. Initialize structured logging
	logger := observability.NewLogger(logLevel, os.Stdout)

	// CLI subcommand: execute migrations and exit
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		var err error
		dbURL := v.GetString("database.url")
		if dbURL == "" {
			logger.Fatal().Msg("database url is required for migrations")
		}
		cfg, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to parse database url")
		}

		// Override connection credentials from standard env vars if set (Helm chart compatibility)
		if dbUser := v.GetString("db.username"); dbUser != "" {
			cfg.ConnConfig.User = dbUser
		}

		if dbPassword := v.GetString("db.password"); dbPassword != "" {
			cfg.ConnConfig.Password = dbPassword
		}

		cfg.ConnConfig.Tracer = observability.NewPgxQueryTracer()

		ctx := context.Background()
		err = keeper.RunMigrations(ctx, cfg, "", logger)
		if err != nil {
			logger.Fatal().Err(err).Msg("migrations failed")
		}
		os.Exit(0)
	}

	logger.Info().
		Str("component", "keeper").
		Str("version", Version).
		Str("port", port).
		Msg("starting component")

	// 3. Initialize OpenTelemetry tracing
	ctx := context.Background()
	shutdownTracer, err := observability.InitTracer(ctx, "keeper", Version, otelEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize tracer")
	}

	// 4. Initialize shutdown manager (30 seconds graceful shutdown timeout)
	mgr := shutdown.NewManager(30 * time.Second)

	// Register tracer shutdown (registered first, runs last)
	mgr.Register("otel-tracer", func(ctx context.Context) error {
		return shutdownTracer(ctx)
	})

	// 5. Initialize PostgreSQL connection pool
	dbURL := v.GetString("database.url")

	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		cfg, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to parse database url")
		}

		// Override connection credentials from standard env vars if set (Helm chart compatibility)
		if dbUser := v.GetString("db.username"); dbUser != "" {
			cfg.ConnConfig.User = dbUser
		}

		if dbPassword := v.GetString("db.password"); dbPassword != "" {
			cfg.ConnConfig.Password = dbPassword
		}

		cfg.ConnConfig.Tracer = observability.NewPgxQueryTracer()

		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to connect to postgres")
		}
		// Register database pool shutdown
		mgr.Register("postgres-pool", func(ctx context.Context) error {
			logger.Info().Msg("closing postgres pool")
			pool.Close()
			return nil
		})
	}

	// 6. Create HTTP router
	router := keeper.NewServer(pool)

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

	// 6. Block until termination signal and execute cleanup
	logger.Info().Msg("component is ready")
	if err := mgr.Wait(syscall.SIGINT, syscall.SIGTERM); err != nil {
		logger.Error().Err(err).Msg("error during graceful shutdown")
		os.Exit(1)
	}

	logger.Info().Msg("component stopped gracefully")
}
