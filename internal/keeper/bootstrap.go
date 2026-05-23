package keeper

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	httpAdapter "github.com/morphy76/tacito-square/internal/keeper/adapters/http"
	"github.com/morphy76/tacito-square/internal/keeper/adapters/postgres"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
)

//go:embed openapi.json
var openapiJSON []byte

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(pool *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.MetricsMiddleware())

	var checkers []health.Checker
	if pool != nil {
		checkers = append(checkers, health.PingChecker("postgres", pool.Ping))
	}

	probe := health.NewProbe(5*time.Second, checkers...)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", openapiJSON)
	})
	r.GET("/metrics", observability.MetricsHandler())

	if pool != nil {
		repo := postgres.NewLLMBindingRepository(pool)
		handler := httpAdapter.NewLLMBindingHandler(repo)

		v1 := r.Group("/api/v1")
		{
			v1.POST("/llm-bindings", handler.Create)
			v1.GET("/llm-bindings", handler.List)
			v1.GET("/llm-bindings/:id", handler.GetByID)
			v1.PUT("/llm-bindings/:id", handler.Update)
			v1.DELETE("/llm-bindings/:id", handler.Delete)
		}
	}

	return r
}
