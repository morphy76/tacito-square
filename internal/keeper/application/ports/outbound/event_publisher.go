package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/pkg/events"
)

// EventPublisher is the driven outbound port for publishing domain events to the message bus.
type EventPublisher interface {
	// Publish publishes a DomainEvent to the message bus for the given subject.
	// Returns an error if publication fails.
	Publish(ctx context.Context, subject string, event events.DomainEvent) error
}
