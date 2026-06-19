package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	var isDomainEvent bool
	if err := json.Unmarshal(data, &evt); err == nil && evt.TenantID != "" {
		isDomainEvent = true
	}

	if isStreamSubject(subject) {
		js, err := p.nc.JetStream()
		if err != nil {
			return fmt.Errorf("failed to get NATS JetStream context: %w", err)
		}

		if isDomainEvent {
			msg := nats.NewMsg(subject)
			msg.Data = data
			msg.Header.Set("X-Tacito-Schema", evt.SchemaRef)
			msg.Header.Set("X-Tacito-Source", evt.Source)
			msg.Header.Set("X-Tacito-Tenant", evt.TenantID)
			msg.Header.Set("X-Tacito-Event-ID", evt.EventID)
			msg.Header.Set("X-Tacito-Occurred", evt.OccurredAt)
			msg.Header.Set("Nats-Msg-Id", evt.EventID)

			observability.InjectNATSContext(ctx, msg)

			if _, err := js.PublishMsg(msg, nats.Context(ctx)); err != nil {
				return fmt.Errorf("failed to publish NATS JetStream message: %w", err)
			}
			return nil
		}

		if _, err := js.Publish(subject, data, nats.Context(ctx)); err != nil {
			return fmt.Errorf("failed to publish NATS JetStream raw data: %w", err)
		}
		return nil
	}

	if isDomainEvent {
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

func isStreamSubject(subject string) bool {
	if !strings.HasPrefix(subject, "ts.community.") {
		return false
	}
	if strings.Contains(subject, ".registry.request") || strings.HasSuffix(subject, ".heartbeat") || strings.HasSuffix(subject, ".status") {
		return false
	}
	return true
}



