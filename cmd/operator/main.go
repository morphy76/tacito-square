package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-logr/zerologr"
	"github.com/morphy76/tacito-square/internal/operator"
	"github.com/morphy76/tacito-square/internal/operator/adapters/inbound"
	"github.com/morphy76/tacito-square/internal/operator/application/service"
	"github.com/morphy76/tacito-square/internal/shared/config"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/shutdown"
	"github.com/morphy76/tacito-square/pkg/kubernetes/apis/tacito/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Version is the application version, set at build time or using fallback.
var Version = "0.1.0"

func main() {
	// 1. Load configuration
	v, err := config.Load("TS_OPERATOR")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	v.SetDefault("port", "8082")

	port := v.GetString("port")
	logLevel := v.GetString("log.level")
	otelEndpoint := v.GetString("otel.endpoint")

	// 2. Initialize structured logging
	logger := observability.NewLogger(logLevel, os.Stdout)
	ctrl.SetLogger(zerologr.New(&logger))

	logger.Info().
		Str("component", "operator").
		Str("version", Version).
		Str("port", port).
		Msg("starting component")

	// 3. Initialize OpenTelemetry tracing
	ctx := context.Background()
	shutdownTracer, err := observability.InitTracer(ctx, "operator", Version, otelEndpoint)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize tracer")
	}

	// 4. Initialize shutdown manager (30 seconds graceful shutdown timeout)
	mgr := shutdown.NewManager(30 * time.Second)

	// Register tracer shutdown (registered first, runs last)
	mgr.Register("otel-tracer", func(ctx context.Context) error {
		return shutdownTracer(ctx)
	})

	// 5. Initialize Kubernetes Client Config and Controller Manager
	k8sCfg, err := k8sconfig.GetConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load kubernetes config")
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Fatal().Err(err).Msg("failed to register client-go scheme")
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		logger.Fatal().Err(err).Msg("failed to register tacito scheme")
	}

	k8sMgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
		Scheme: scheme,
		Logger: zerologr.New(&logger),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize controller manager")
	}

	// 6. Initialize Reconciler Inbound Adapter and Application Service
	reconcileService := service.NewReconcileAgentService(k8sMgr.GetClient(), logger, v)
	reconciler := inbound.NewTacitoAgentReconciler(k8sMgr.GetClient(), scheme, reconcileService, logger)
	if err := reconciler.SetupWithManager(k8sMgr); err != nil {
		logger.Fatal().Err(err).Msg("failed to setup reconciler with manager")
	}

	// Start Controller Manager in background
	mgrCtx, cancelMgr := context.WithCancel(context.Background())
	go func() {
		logger.Info().Msg("starting controller manager")
		if err := k8sMgr.Start(mgrCtx); err != nil {
			logger.Error().Err(err).Msg("controller manager stopped with error")
		}
	}()

	// Register manager shutdown
	mgr.Register("controller-manager", func(ctx context.Context) error {
		logger.Info().Msg("stopping controller manager")
		cancelMgr()
		return nil
	})

	// 7. Create HTTP router with readiness/liveness probes
	kubeAPIChecker := operator.KubeAPIChecker(k8sMgr.GetAPIReader())
	router := operator.NewServer(kubeAPIChecker)

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

	// 8. Block until termination signal and execute cleanup
	logger.Info().Msg("component is ready")
	if err := mgr.Wait(syscall.SIGINT, syscall.SIGTERM); err != nil {
		logger.Error().Err(err).Msg("error during graceful shutdown")
		os.Exit(1)
	}

	logger.Info().Msg("component stopped gracefully")
}
