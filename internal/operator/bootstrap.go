package operator

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(checkers ...health.Checker) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	probe := health.NewProbe(5*time.Second, checkers...)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/metrics", observability.MetricsHandler())

	return r
}

// KubeAPIChecker creates a health.Checker that lists namespaces to verify Kubernetes API server connectivity.
func KubeAPIChecker(c client.Client) health.Checker {
	return health.Checker{
		Name: "kubernetes-api",
		Check: func(ctx context.Context) error {
			var nsList corev1.NamespaceList
			if err := c.List(ctx, &nsList, client.Limit(1)); err != nil {
				return err
			}
			return nil
		},
	}
}
