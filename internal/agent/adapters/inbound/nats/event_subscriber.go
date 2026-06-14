package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	role        string
	router      inbound.SchemaRouter
	blobStore   outbound.BlobStore
	logger      zerolog.Logger
	subs        []*natsclient.Subscription
}

func NewEventSubscriber(
	nc *natsclient.Conn,
	agentName string,
	communityID string,
	role string,
	router inbound.SchemaRouter,
	blobStore outbound.BlobStore,
	logger zerolog.Logger,
) *EventSubscriber {
	return &EventSubscriber{
		nc:          nc,
		agentName:   agentName,
		communityID: communityID,
		role:        role,
		router:      router,
		blobStore:   blobStore,
		logger:      logger,
	}
}

func (s *EventSubscriber) Start(ctx context.Context) error {
	s.subs = nil

	if s.role == "hub" {
		hubSubj := fmt.Sprintf("ts.community.%s.agent.hub", s.communityID)
		queueGroup := fmt.Sprintf("hub-queue-group-%s", s.communityID)
		subHub, err := s.nc.QueueSubscribe(hubSubj, queueGroup,
			observability.WrapNATSHandler("nats.event_handler", s.logger, s.handleEvent))
		if err != nil {
			return fmt.Errorf("event subscriber: queue subscribe to %s: %w", hubSubj, err)
		}
		s.subs = append(s.subs, subHub)

		// Hub also subscribes to Spoke responses in the community
		respSubj := fmt.Sprintf("ts.community.%s.agent.*.thread.*.response", s.communityID)
		respQueueGroup := fmt.Sprintf("hub-queue-group-response-%s", s.communityID)
		subResp, err := s.nc.QueueSubscribe(respSubj, respQueueGroup,
			observability.WrapNATSHandler("nats.event_handler", s.logger, s.handleEvent))
		if err != nil {
			_ = subHub.Unsubscribe()
			s.subs = nil
			return fmt.Errorf("event subscriber: queue subscribe to %s: %w", respSubj, err)
		}
		s.subs = append(s.subs, subResp)

		s.logger.Info().
			Str("hub_subject", hubSubj).
			Str("response_subject", respSubj).
			Str("queue_group", queueGroup).
			Msg("event subscriber started as hub")
	} else {
		// 1. Subscribe to specific agent subject
		agentSubj := fmt.Sprintf("ts.community.%s.agent.%s", s.communityID, s.agentName)
		subAgent, err := s.nc.Subscribe(agentSubj,
			observability.WrapNATSHandler("nats.event_handler", s.logger, s.handleEvent))
		if err != nil {
			return fmt.Errorf("event subscriber: subscribe to %s: %w", agentSubj, err)
		}
		s.subs = append(s.subs, subAgent)

		// 2. Subscribe to community broadcast subject
		allSubj := fmt.Sprintf("ts.community.%s.agent.all", s.communityID)
		subAll, err := s.nc.Subscribe(allSubj,
			observability.WrapNATSHandler("nats.event_handler", s.logger, s.handleEvent))
		if err != nil {
			_ = subAgent.Unsubscribe()
			s.subs = nil
			return fmt.Errorf("event subscriber: subscribe to %s: %w", allSubj, err)
		}
		s.subs = append(s.subs, subAll)

		s.logger.Info().
			Str("agent_subject", agentSubj).
			Str("broadcast_subject", allSubj).
			Msg("event subscriber started as spoke")
	}
	return nil
}

func (s *EventSubscriber) Stop() error {
	var errs []string
	for _, sub := range s.subs {
		if sub != nil {
			if err := sub.Drain(); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	s.subs = nil
	if len(errs) > 0 {
		return fmt.Errorf("failed to drain some subscriptions: %s", strings.Join(errs, ", "))
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
