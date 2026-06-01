package outbound

import "context"

// EventPublisher defines the port for publishing event payloads asynchronously.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}
