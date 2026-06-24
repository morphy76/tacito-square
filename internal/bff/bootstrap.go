package bff

import (
	"context"
	"crypto/sha256"
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

//go:embed secure/index.html
var secureIndexHTML []byte

//go:embed favicon.ico
var faviconICO []byte

// Config holds configuration parameters for the BFF server bootstrap.
type Config struct {
	Version      string
	OtelEndpoint string
	LogLevel     string
	GinMode      string
	UIPath       string
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

	// Precompute ETags for static resources
	welcomeHash := sha256.Sum256(welcomeHTML)
	welcomeETag := fmt.Sprintf(`"%x"`, welcomeHash)

	secureHash := sha256.Sum256(secureIndexHTML)
	secureETag := fmt.Sprintf(`"%x"`, secureHash)

	faviconHash := sha256.Sum256(faviconICO)
	faviconETag := fmt.Sprintf(`"%x"`, faviconHash)

	// Public system endpoints (metrics and health check probes)
	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/metrics", observability.MetricsHandler())

	// Serve OIDC / OpenAPI spec
	var finalOpenAPI []byte
	if cfg.Version != "" {
		finalOpenAPI = []byte(strings.ReplaceAll(string(openapiJSON), `"version": "0.1.0"`, fmt.Sprintf(`"version": "%s"`, cfg.Version)))
	} else {
		finalOpenAPI = openapiJSON
	}

	// Precompute ETag for OpenAPI JSON
	hash := sha256.Sum256(finalOpenAPI)
	etag := fmt.Sprintf(`"%x"`, hash)

	faviconHandler := func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=604800")
		c.Header("ETag", faviconETag)
		if c.GetHeader("If-None-Match") == faviconETag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "image/x-icon", faviconICO)
	}

	welcomeHandler := func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=0, must-revalidate")
		c.Header("ETag", welcomeETag)
		if c.GetHeader("If-None-Match") == welcomeETag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", welcomeHTML)
	}

	secureHandler := func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=0, must-revalidate")
		c.Header("ETag", secureETag)
		if c.GetHeader("If-None-Match") == secureETag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", secureIndexHTML)
	}

	// UI Static and Welcome Homepage Route Group
	uiGroup := r.Group(cfg.UIPath)
	{
		uiGroup.GET("", func(c *gin.Context) {
			if !strings.HasSuffix(c.Request.URL.Path, "/") {
				target := c.Request.URL.Path + "/"
				if c.Request.URL.RawQuery != "" {
					target += "?" + c.Request.URL.RawQuery
				}
				c.Redirect(http.StatusMovedPermanently, target)
				return
			}
			welcomeHandler(c)
		})
		uiGroup.GET("/favicon.ico", faviconHandler)
		uiGroup.GET("/", welcomeHandler)
		uiGroup.GET("/index.html", welcomeHandler)
		uiGroup.GET("/openapi.json", func(c *gin.Context) {
			c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
			c.Header("ETag", etag)
			if c.GetHeader("If-None-Match") == etag {
				c.Status(http.StatusNotModified)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", finalOpenAPI)
		})

		// Secure Welcome Homepage with OIDC redirect auth under UI path
		secureGroup := uiGroup.Group("/secure")
		secureGroup.Use(httpAdapter.AuthRedirectMiddleware(sessionUC, cfg.UIPath))
		{
			secureGroup.GET("", secureHandler)
			secureGroup.GET("/", secureHandler)
			secureGroup.GET("/index.html", secureHandler)
		}
	}

	r.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
		c.Header("ETag", etag)
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", finalOpenAPI)
	})

	// Register application specific routes
	httpAdapter.RegisterRoutes(r, sessionUC, eventUC, cfg.UIPath)

	return r
}
