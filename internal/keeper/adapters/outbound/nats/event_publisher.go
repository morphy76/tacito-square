package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/pkg/events"
	natsclient "github.com/nats-io/nats.go"
)

type NATSEventPublisher struct {
	nc *natsclient.Conn
}

func NewNATSEventPublisher(nc *natsclient.Conn) *NATSEventPublisher {
	return &NATSEventPublisher{nc: nc}
}

func (p *NATSEventPublisher) Publish(ctx context.Context, subject string, event events.DomainEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := natsclient.NewMsg(subject)
	msg.Data = data

	// Project headers
	msg.Header.Set("X-Tacito-Schema", event.SchemaRef)
	msg.Header.Set("X-Tacito-Source", event.Source)
	msg.Header.Set("X-Tacito-Tenant", event.TenantID)
	msg.Header.Set("X-Tacito-Event-ID", event.EventID)
	msg.Header.Set("X-Tacito-Occurred", event.OccurredAt)

	// Inject trace context
	observability.InjectNATSContext(ctx, msg)

	if isStreamSubject(subject) {
		js, err := p.nc.JetStream()
		if err != nil {
			return fmt.Errorf("failed to get NATS JetStream context: %w", err)
		}
		// Set native Nats-Msg-Id for deduplication
		msg.Header.Set("Nats-Msg-Id", event.EventID)

		if _, err := js.PublishMsg(msg, natsclient.Context(ctx)); err != nil {
			return fmt.Errorf("failed to publish NATS JetStream message: %w", err)
		}
		return nil
	}

	if err := p.nc.PublishMsg(msg); err != nil {
		return fmt.Errorf("failed to publish NATS message: %w", err)
	}

	return nil
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


