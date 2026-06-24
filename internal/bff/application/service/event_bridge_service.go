package service

import (
	"context"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

// EventBridgeService acts as a driving adapter client bridge for SSE event logs.
type EventBridgeService struct {
	client outbound.BackendEventClient
}

// NewEventBridgeService constructs a new EventBridgeService.
func NewEventBridgeService(client outbound.BackendEventClient) *EventBridgeService {
	return &EventBridgeService{
		client: client,
	}
}

// StreamEvents proxies event streaming requests directly to the downstream BackendEventClient port.
func (s *EventBridgeService) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	return s.client.StreamEvents(ctx, tenantID)
}
