package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/pkg/events"
)

// EventSubscription represents an active subscription that can be stopped.
type EventSubscription interface {
	Stop() error
}

// EventSubscriber is the driven outbound port for subscribing to domain event streams.
type EventSubscriber interface {
	// Subscribe creates a subscription on the given subject pattern.
	// Events matching tenantID are forwarded to the handler.
	// Returns an EventSubscription that the caller MUST Stop() on cleanup.
	Subscribe(ctx context.Context, subjectPattern string, tenantID string, handler func(*events.DomainEvent)) (EventSubscription, error)
}
