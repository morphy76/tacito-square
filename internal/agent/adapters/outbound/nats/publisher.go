package nats

import (
	"context"
	"encoding/json"

	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/pkg/events"
	"github.com/nats-io/nats.go"
)

type NATSEventPublisher struct {
	nc *nats.Conn
}

func NewNATSEventPublisher(nc *nats.Conn) *NATSEventPublisher {
	return &NATSEventPublisher{nc: nc}
}

func (p *NATSEventPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	var evt events.DomainEvent
	if err := json.Unmarshal(data, &evt); err == nil && evt.TenantID != "" {
		msg := nats.NewMsg(subject)
		msg.Data = data

		msg.Header.Set("X-Tacito-Schema", evt.SchemaRef)
		msg.Header.Set("X-Tacito-Source", evt.Source)
		msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
		msg.Header.Set("X-Tacito-Event-ID", evt.EventID)
		msg.Header.Set("X-Tacito-Occurred", evt.OccurredAt)

		observability.InjectNATSContext(ctx, msg)

		return p.nc.PublishMsg(msg)
	}

	return p.nc.Publish(subject, data)
}

