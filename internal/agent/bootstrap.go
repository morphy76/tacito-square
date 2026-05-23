package agent

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/shared/health"
)

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	// Hello world health check probe without additional checkers.
	probe := health.NewProbe(5 * time.Second)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))

	return r
}
