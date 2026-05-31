package inbound

import "context"

// MessageProcessor defines the driving port for the message processing pipeline.
type MessageProcessor interface {
	ProcessIncomingMessage(ctx context.Context, payload string) (string, error)
}
