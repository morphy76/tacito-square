package outbound

import "context"

// BackendEventClient defines the outbound port for streaming event logs from the backend.
type BackendEventClient interface {
	// StreamEvents establishes a stream of raw JSON event messages from the backend for the specified tenant.
	StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error)
}
