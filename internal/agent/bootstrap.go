package agent

import (
	"time"

	"github.com/gin-gonic/gin"
	agentnats "github.com/morphy76/tacito-square/internal/agent/adapters/inbound/nats"
	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/shared/health"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// EventSubscriber is a re-export of the NATS event subscriber for use in the entrypoint.
type EventSubscriber = agentnats.EventSubscriber

// NewEventSubscriber constructs a new EventSubscriber for the given agent identity.
func NewEventSubscriber(nc *natsclient.Conn, agentName, communityID, role string, router inbound.SchemaRouter, blobStore outbound.BlobStore, logger zerolog.Logger) *EventSubscriber {
	return agentnats.NewEventSubscriber(nc, agentName, communityID, role, router, blobStore, logger)
}

// ConnectNATS establishes a NATS connection with default reconnect options and logs the outcome.
func ConnectNATS(url string, logger zerolog.Logger) (*natsclient.Conn, error) {
	nc, err := natsclient.Connect(
		url,
		natsclient.MaxReconnects(-1),
		natsclient.ReconnectWait(2*time.Second),
		natsclient.DisconnectErrHandler(func(_ *natsclient.Conn, err error) {
			if err != nil {
				logger.Warn().Err(err).Msg("NATS disconnected")
			}
		}),
		natsclient.ReconnectHandler(func(_ *natsclient.Conn) {
			logger.Trace().Msg("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, err
	}
	logger.Debug().Str("nats.url", url).Msg("NATS connection established")
	return nc, nil
}

// NewServer creates and configures a new Gin HTTP server with health probes.
func NewServer(checkers ...health.Checker) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability.TracingMiddleware("agent"))
	r.Use(observability.MetricsMiddleware())
	r.Use(observability.LoggingMiddleware())

	// Hello world health check probe with optional dependency checkers.
	probe := health.NewProbe(5 * time.Second, checkers...)

	r.GET("/healthz", gin.WrapF(probe.LivezHandler))
	r.GET("/readyz", gin.WrapF(probe.ReadyzHandler))
	r.GET("/metrics", observability.MetricsHandler())

	return r
}

