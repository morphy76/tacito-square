package inbound

import "context"

// EventStreamUseCase defines the inbound port (driving interface) for streaming events to clients.
type EventStreamUseCase interface {
	// StreamEvents establishes an event stream for client consumers.
	StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error)
}
