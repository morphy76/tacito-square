package nats

import (
	"context"

	"github.com/nats-io/nats.go"
)

type NATSEventPublisher struct {
	nc *nats.Conn
}

func NewNATSEventPublisher(nc *nats.Conn) *NATSEventPublisher {
	return &NATSEventPublisher{nc: nc}
}

func (p *NATSEventPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	return p.nc.Publish(subject, data)
}
