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

//go:embed secure/index.html
var secureIndexHTML []byte

//go:embed bootstrap-theme.css
var bootstrapThemeCSS []byte

//go:embed favicon.ico
var faviconICO []byte

// Config holds configuration parameters for the BFF server bootstrap.
type Config struct {
	Version           string
	OtelEndpoint      string
	LogLevel          string
	GinMode           string
	UIPath            string
	UIConfiguratorURL string
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



	secureHash := sha256.Sum256(secureIndexHTML)
	secureETag := fmt.Sprintf(`"%x"`, secureHash)

	faviconHash := sha256.Sum256(faviconICO)
	faviconETag := fmt.Sprintf(`"%x"`, faviconHash)

	themeHash := sha256.Sum256(bootstrapThemeCSS)
	themeETag := fmt.Sprintf(`"%x"`, themeHash)

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


	secureHandler := func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=0, must-revalidate")
		c.Header("ETag", secureETag)
		if c.GetHeader("If-None-Match") == secureETag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", secureIndexHTML)
	}

	uiProxy := httpAdapter.NewUIProxyHandler(store, cfg.UIConfiguratorURL)

	// UI Static and Welcome Homepage Route Group
	uiGroup := r.Group(cfg.UIPath)
	{
		uiGroup.GET("", func(c *gin.Context) {
			target := c.Request.URL.Path + "/"
			if c.Request.URL.RawQuery != "" {
				target += "?" + c.Request.URL.RawQuery
			}
			c.Redirect(http.StatusMovedPermanently, target)
		})

		themeHandler := func(c *gin.Context) {
			c.Header("Cache-Control", "public, max-age=604800, must-revalidate")
			c.Header("ETag", themeETag)
			if c.GetHeader("If-None-Match") == themeETag {
				c.Status(http.StatusNotModified)
				return
			}
			c.Data(http.StatusOK, "text/css; charset=utf-8", bootstrapThemeCSS)
		}

		uiGroup.GET("/favicon.ico", faviconHandler)
		uiGroup.GET("/assets/bootstrap-theme.css", themeHandler)
		uiGroup.GET("/openapi.json", func(c *gin.Context) {
			c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
			c.Header("ETag", etag)
			if c.GetHeader("If-None-Match") == etag {
				c.Status(http.StatusNotModified)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", finalOpenAPI)
		})

		// Secure UI SPA routes protected by OIDC AuthRedirectMiddleware
		uiSecureGroup := uiGroup.Group("")
		uiSecureGroup.Use(httpAdapter.AuthRedirectMiddleware(sessionUC, cfg.UIPath))
		{
			uiSecureGroup.GET("/", uiProxy.ServeUIIndex)
			uiSecureGroup.GET("/index.html", uiProxy.ServeUIIndex)

			// Keep /secure for backward compatibility/verification
			uiSecureGroup.GET("/secure", secureHandler)
			uiSecureGroup.GET("/secure/", secureHandler)
			uiSecureGroup.GET("/secure/index.html", secureHandler)
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
	httpAdapter.RegisterRoutes(r, sessionUC, eventUC, keeperClient, cfg.UIPath)

	// SPA fallback routing for any non-matched paths under UIPath
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, cfg.UIPath) {
			httpAdapter.AuthRedirectMiddleware(sessionUC, cfg.UIPath)(c)
			if !c.IsAborted() {
				uiProxy.ServeUIIndex(c)
			}
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	return r
}
