package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/events"
	natsclient "github.com/nats-io/nats.go"
)

type NATSEventSubscriber struct {
	nc *natsclient.Conn
}

func NewNATSEventSubscriber(nc *natsclient.Conn) *NATSEventSubscriber {
	return &NATSEventSubscriber{nc: nc}
}

type natsSubscription struct {
	sub *natsclient.Subscription
}

func (s *natsSubscription) Stop() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

func (s *NATSEventSubscriber) Subscribe(ctx context.Context, subjectPattern string, tenantID string, handler func(*events.DomainEvent)) (outbound.EventSubscription, error) {
	sub, err := s.nc.Subscribe(subjectPattern, func(msg *natsclient.Msg) {
		tenantHeader := msg.Header.Get("X-Tacito-Tenant")
		if tenantHeader != tenantID {
			return // tenant isolation check
		}

		var evt events.DomainEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			// Skip malformed payloads gracefully
			return
		}

		handler(&evt)
	})
	if err != nil {
		return nil, fmt.Errorf("failed NATS subscribe: %w", err)
	}

	return &natsSubscription{sub: sub}, nil
}
