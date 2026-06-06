package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/morphy76/tacito-square/pkg/events"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

type EventSubscriber struct {
	nc          *natsclient.Conn
	agentName   string
	communityID string
	router      inbound.SchemaRouter
	blobStore   outbound.BlobStore
	logger      zerolog.Logger
	sub         *natsclient.Subscription
}

func NewEventSubscriber(
	nc *natsclient.Conn,
	agentName string,
	communityID string,
	router inbound.SchemaRouter,
	blobStore outbound.BlobStore,
	logger zerolog.Logger,
) *EventSubscriber {
	return &EventSubscriber{
		nc:          nc,
		agentName:   agentName,
		communityID: communityID,
		router:      router,
		blobStore:   blobStore,
		logger:      logger,
	}
}

func (s *EventSubscriber) Start(ctx context.Context) error {
	subject := fmt.Sprintf("ts.community.%s.agent.%s", s.communityID, s.agentName)
	sub, err := s.nc.Subscribe(subject,
		observability.WrapNATSHandler("nats.event_handler", s.logger, s.handleEvent))
	if err != nil {
		return fmt.Errorf("event subscriber: subscribe to %s: %w", subject, err)
	}
	s.sub = sub
	s.logger.Info().Str("subject", subject).Msg("event subscriber started")
	return nil
}

func (s *EventSubscriber) Stop() error {
	if s.sub != nil {
		err := s.sub.Drain()
		s.sub = nil
		return err
	}
	return nil
}

func (s *EventSubscriber) handleEvent(ctx context.Context, logger zerolog.Logger, msg *natsclient.Msg) error {
	var evt events.DomainEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		logger.Warn().Err(err).Msg("event subscriber: malformed payload, ignoring")
		return nil
	}

	schemaHeader := msg.Header.Get("X-Tacito-Schema")
	if schemaHeader == "" {
		schemaHeader = evt.SchemaRef
	}
	if schemaHeader == "" {
		logger.Warn().Msg("event subscriber: missing schema header, dropping")
		return nil
	}

	// Payload offloading check for add-user-message
	if evt.SchemaRef == events.SchemaConversationalAddUserMessage && s.blobStore != nil {
		var payload events.AddUserMessagePayload
		if err := json.Unmarshal(evt.Payload, &payload); err == nil {
			if len(payload.Message) > 256*1024 {
				// Offload payload.Message to S3
				refStr, err := OffloadPayload(ctx, s.blobStore, s.communityID, s.agentName, payload.ThreadID, evt.TenantID, []byte(payload.Message))
				if err != nil {
					logger.Error().Err(err).Msg("failed to offload large message payload to S3")
					return err
				}
				// Replace message with reference string
				payload.Message = refStr
				newPayloadBytes, err := json.Marshal(payload)
				if err == nil {
					evt.Payload = newPayloadBytes
				}
			}
		}
	}

	ctx = logger.WithContext(ctx)

	if err := s.router.RouteEvent(ctx, evt); err != nil {
		logger.Error().Err(err).Msg("failed to route event through schema router")
		return err
	}

	return nil
}
