package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/tacito-square/internal/keeper/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/tenant"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/attribute"
)

type EventHandler struct {
	eventUseCase       inbound.EventUseCase
	eventStreamUseCase inbound.EventStreamUseCase
}

func NewEventHandler(eventUseCase inbound.EventUseCase, eventStreamUseCase inbound.EventStreamUseCase) *EventHandler {
	return &EventHandler{
		eventUseCase:       eventUseCase,
		eventStreamUseCase: eventStreamUseCase,
	}
}

type publishRequest struct {
	SchemaRef string          `json:"schema_ref" binding:"required"`
	Payload   json.RawMessage `json:"payload" binding:"required"`
}

func (h *EventHandler) PublishEvent(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	_, span := observability.StartHandlerSpan(c, "keeper.publish_event")

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	var req publishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	span.SetAttributes(
		attribute.String("schema_ref", req.SchemaRef),
		attribute.String("tenant_id", ten.FullName()),
	)

	observability.SpanEventFromGinContext(c, "keeper.request.accepted",
		attribute.String("schema_ref", req.SchemaRef),
	)

	evt, err := h.eventUseCase.PublishEvent(ctx, req.SchemaRef, req.Payload)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "invalid schema_ref") {
			status = http.StatusUnprocessableEntity
		} else if strings.Contains(err.Error(), "unauthorized") {
			status = http.StatusForbidden
		}

		observability.RecordGinError(c, status, err, attribute.String("schema_ref", req.SchemaRef))
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	reqLogger.Info().
		Str("event_id", evt.EventID).
		Str("schema_ref", evt.SchemaRef).
		Str("tenant_id", ten.FullName()).
		Msg("event published successfully")

	if evt.SchemaRef == events.SchemaConversationalStartThread {
		var stPayload events.StartThreadPayload
		if err := json.Unmarshal(evt.Payload, &stPayload); err == nil {
			c.Header("Location", fmt.Sprintf("/api/v1/communities/%s/threads/%s", stPayload.CommunityID, stPayload.ThreadID))
		}
	}

	c.JSON(http.StatusAccepted, evt)
}

var sseSchemaBlacklist = map[string]bool{
	events.SchemaInfrastructureAgentHeartbeat: true,
}

func (h *EventHandler) StreamEvents(c *gin.Context) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	logger := observability.NewLogger("info", os.Stdout)
	reqLogger := observability.WithContext(logger, ctx)

	ten := tenant.FromContext(ctx)
	if ten == nil {
		reqLogger.Warn().Msg("unauthorized: missing tenant context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant is required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	reqLogger.Info().
		Str("tenant_id", ten.FullName()).
		Msg("client connected to event stream")

	// Get heartbeat config
	heartbeatSec := viper.GetInt("keeper.sse.heartbeat_seconds")
	if heartbeatSec <= 0 {
		heartbeatSec = 30
	}
	heartbeatInterval := time.Duration(heartbeatSec) * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// Channel to receive SSE events to write
	eventChan := make(chan string, 100)

	sub, err := h.eventStreamUseCase.SubscribeEvents(ctx, ten.FullName(), func(evt *events.DomainEvent) {
		if evt.SchemaRef == "" || sseSchemaBlacklist[evt.SchemaRef] {
			return
		}

		// Format: event is last segment of schema URN (e.g. "urn:tacito:schema:conversational:start-thread:v1" -> "start-thread")
		parts := strings.Split(evt.SchemaRef, ":")
		eventType := parts[len(parts)-1]
		if eventType == "v1" && len(parts) >= 2 {
			eventType = parts[len(parts)-2]
		}

		dataBytes, err := json.Marshal(evt)
		if err != nil {
			return
		}

		sseMessage := fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", evt.EventID, eventType, string(dataBytes))
		select {
		case eventChan <- sseMessage:
		default:
			// drop if full to avoid blocking
		}
	})
	if err != nil {
		reqLogger.Error().Err(err).Msg("failed to subscribe to NATS wildcard events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer sub.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			reqLogger.Info().Msg("client disconnected from event stream")
			return false
		case msg := <-eventChan:
			_, _ = w.Write([]byte(msg))
			return true
		case <-heartbeatTicker.C:
			_, _ = w.Write([]byte(": keepalive\n\n"))
			return true
		}
	})
}
