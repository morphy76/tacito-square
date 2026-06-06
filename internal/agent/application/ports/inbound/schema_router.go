package inbound

import (
	"context"

	"github.com/morphy76/tacito-square/pkg/events"
)

// SchemaRouter defines the interface for routing domain events based on their schemaRef.
type SchemaRouter interface {
	RouteEvent(ctx context.Context, event events.DomainEvent) error
}
