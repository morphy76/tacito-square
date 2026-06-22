package bff

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	httpAdapter "github.com/morphy76/tacito-square/internal/bff/adapters/inbound/http"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
)

//go:embed bff_openapi.json
var openapiJSON []byte

//go:embed index.html
var welcomeHTML []byte

// Config holds configuration parameters for the BFF server bootstrap.
type Config struct {
	Version      string
	OtelEndpoint string
	LogLevel     string
	GinMode      string
}

// Pinger defines a simple ping interface for health checking.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewServer creates and configures a new Gin HTTP server with health probes, tracing, metrics, and routing.
func NewServer(
	cfg Config,
	sessionUC inbound.SessionUseCase,
	eventUC inbound.EventStreamUseCase,
	store outbound.SessionStore,
	oidcProvider outbound.OIDCProvider,
	keeperClient outbound.KeeperClient,
) *gin.Engine {
	// Initialize structured logging using zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(os.Stdout)
	if parsedLevel, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		log.Logger = log.Logger.Level(parsedLevel)
	} else {
		log.Logger = log.Logger.Level(zerolog.InfoLevel)
	}

	log.Info().
		Str("component", "bff").
		Str("version", cfg.Version).
		Msg("starting bff component")

	if cfg.GinMode == string(gin.ReleaseMode) {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Wire OpenTelemetry tracing and Prometheus metrics middlewares
	r.Use(observability.TracingMiddleware("bff"))
	r.Use(observability.MetricsMiddleware())

	// Setup health check dependencies (Option A: check only Redis to avoid cascading HTTP checks)
	var checkers []health.Checker
	if store != nil {
		if pinger, ok := store.(Pinger); ok {
			checkers = append(checkers, health.Checker{
				Name: "redis",
				Check: func(ctx context.Context) error {
					return pinger.Ping(ctx)
				},
			})
		}
	}

	probe := health.NewProbe(5*time.Second, checkers...)

	// Public system endpoints (metrics and health check probes)
	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/metrics", observability.MetricsHandler())

	// Serve Welcome Homepage
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", welcomeHTML)
	})
	r.GET("/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", welcomeHTML)
	})

	// Serve OIDC / OpenAPI spec
	var finalOpenAPI []byte
	if cfg.Version != "" {
		finalOpenAPI = []byte(strings.ReplaceAll(string(openapiJSON), `"version": "0.1.0"`, fmt.Sprintf(`"version": "%s"`, cfg.Version)))
	} else {
		finalOpenAPI = openapiJSON
	}

	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", finalOpenAPI)
	})


	// Register application specific routes
	httpAdapter.RegisterRoutes(r, sessionUC, eventUC)

	return r
}
