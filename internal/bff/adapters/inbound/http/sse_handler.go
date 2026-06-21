package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
)

type SSEHandler struct {
	eventUC inbound.EventStreamUseCase
}

func NewSSEHandler(eventUC inbound.EventStreamUseCase) *SSEHandler {
	return &SSEHandler{eventUC: eventUC}
}

func (h *SSEHandler) StreamEvents(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	tenantID, ok := tenantIDVal.(string)
	if !ok || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	ch, err := h.eventUC.StreamEvents(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	heartbeatSeconds := viper.GetInt("bff.sse.heartbeat_seconds")
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = 30
	}

	ticker := time.NewTicker(time.Duration(heartbeatSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(event))
			c.Writer.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprint(c.Writer, ": keep-alive\n\n")
			c.Writer.Flush()
		}
	}
}
