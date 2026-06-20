package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

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

	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
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
		logger:      logger.With().Str("component", "event_subscriber").Logger(),
	}
}

func (s *EventSubscriber) Start(ctx context.Context) error {
	s.subs = nil
	s.ctx, s.cancel = context.WithCancel(ctx)

	js, err := s.nc.JetStream()
	if err != nil {
		return fmt.Errorf("event subscriber: get JetStream context: %w", err)
	}

	// Assert streams
	streamCfg := &natsclient.StreamConfig{
		Name:     "TACITO_EVENTS",
		Subjects: []string{
			"ts.community.*.agent.*",
			"ts.community.*.agent.*.thread.*.response",
			"ts.community.*.agent.*.thread.*.history",
		},
		MaxAge:   7 * 24 * time.Hour,
	}
	_, err = js.AddStream(streamCfg)
	if err != nil {
		if strings.Contains(err.Error(), "stream name already in use") {
			if _, updateErr := js.UpdateStream(streamCfg); updateErr != nil {
				s.logger.Warn().Err(updateErr).Msg("failed to update TACITO_EVENTS stream configuration, proceeding with existing stream")
			}
		} else {
			return fmt.Errorf("event subscriber: create TACITO_EVENTS stream: %w", err)
		}
	}

	dlqCfg := &natsclient.StreamConfig{
		Name:     "TACITO_DLQ",
		Subjects: []string{"ts.dlq.community.>"},
		MaxAge:   7 * 24 * time.Hour,
	}
	_, err = js.AddStream(dlqCfg)
	if err != nil {
		if strings.Contains(err.Error(), "stream name already in use") {
			if _, updateErr := js.UpdateStream(dlqCfg); updateErr != nil {
				s.logger.Warn().Err(updateErr).Msg("failed to update TACITO_DLQ stream configuration, proceeding with existing stream")
			}
		} else {
			return fmt.Errorf("event subscriber: create TACITO_DLQ stream: %w", err)
		}
	}

	if s.role == "hub" {
		hubSubj := fmt.Sprintf("ts.community.%s.agent.hub", s.communityID)
		durableHub := fmt.Sprintf("hub-subscriber-%s", s.communityID)
		if err := s.startPullLoop(s.ctx, js, hubSubj, durableHub); err != nil {
			s.cancel()
			return err
		}

		respSubj := fmt.Sprintf("ts.community.%s.agent.*.thread.*.response", s.communityID)
		durableResp := fmt.Sprintf("hub-response-subscriber-%s", s.communityID)
		if err := s.startPullLoop(s.ctx, js, respSubj, durableResp); err != nil {
			s.cancel()
			return err
		}
	} else {
		agentSubj := fmt.Sprintf("ts.community.%s.agent.%s", s.communityID, s.agentName)
		durableAgent := fmt.Sprintf("spoke-subscriber-%s-%s", s.communityID, s.agentName)
		if err := s.startPullLoop(s.ctx, js, agentSubj, durableAgent); err != nil {
			s.cancel()
			return err
		}

		allSubj := fmt.Sprintf("ts.community.%s.agent.all", s.communityID)
		durableAll := fmt.Sprintf("spoke-broadcast-subscriber-%s-%s", s.communityID, s.agentName)
		if err := s.startPullLoop(s.ctx, js, allSubj, durableAll); err != nil {
			s.cancel()
			return err
		}
	}

	return nil
}

func (s *EventSubscriber) startPullLoop(ctx context.Context, js natsclient.JetStreamContext, subject, durableName string) error {
	// Create durable consumer explicitly to prevent auto-deletion on Unsubscribe
	consumerCfg := &natsclient.ConsumerConfig{
		Durable:       durableName,
		FilterSubject: subject,
		AckPolicy:     natsclient.AckExplicitPolicy,
		MaxWaiting:    128,
	}
	_, err := js.AddConsumer("TACITO_EVENTS", consumerCfg)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "consumer name already in use") ||
			strings.Contains(errStr, "already exists") {
			// Configuration might be different, let's update it
			if _, updateErr := js.UpdateConsumer("TACITO_EVENTS", consumerCfg); updateErr != nil {
				s.logger.Warn().Err(updateErr).Str("consumer", durableName).Msg("failed to update consumer configuration, proceeding with existing consumer")
			}
		} else {
			return fmt.Errorf("failed to create durable consumer %s: %w", durableName, err)
		}
	}

	// Bind to the pre-existing durable pull consumer
	sub, err := js.PullSubscribe(subject, "", natsclient.Bind("TACITO_EVENTS", durableName), natsclient.PullMaxWaiting(128))
	if err != nil {
		return fmt.Errorf("pull subscribe binding to durable %s failed: %w", durableName, err)
	}
	s.subs = append(s.subs, sub)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Info().Str("subject", subject).Str("durable", durableName).Msg("started pull consumer worker loop")

		for {
			select {
			case <-ctx.Done():
				s.logger.Debug().Str("subject", subject).Msg("stopping pull consumer worker loop (context cancelled)")
				return
			default:
				// Fetch 1 message at a time with a short timeout to check loop context cancellation
				fetchCtx, fetchCancel := context.WithTimeout(ctx, 1*time.Second)
				msgs, err := sub.Fetch(1, natsclient.Context(fetchCtx))
				fetchCancel()

				if err != nil {
					if err == natsclient.ErrTimeout || err == context.DeadlineExceeded || ctx.Err() != nil {
						continue
					}
					s.logger.Warn().Err(err).Str("subject", subject).Msg("error fetching from JetStream pull consumer")
					time.Sleep(500 * time.Millisecond) // backoff on connection/stream issues
					continue
				}

				for _, msg := range msgs {
					s.processAndAck(ctx, js, msg)
				}
			}
		}
	}()

	return nil
}

func (s *EventSubscriber) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()

	var errs []string
	for _, sub := range s.subs {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	s.subs = nil

	if len(errs) > 0 {
		return fmt.Errorf("failed to unsubscribe pull consumers: %s", strings.Join(errs, ", "))
	}
	return nil
}

func (s *EventSubscriber) processAndAck(ctx context.Context, js natsclient.JetStreamContext, msg *natsclient.Msg) {
	ctx = observability.ExtractNATSContext(ctx, msg)
	logger := observability.WithContext(s.logger, ctx).With().
		Str("subject", msg.Subject).
		Logger()
	ctx = logger.WithContext(ctx)

	// Check delivery count for DLQ routing
	meta, err := msg.Metadata()
	if err == nil && meta.NumDelivered > 5 {
		logger.Warn().Uint64("num_delivered", meta.NumDelivered).Msg("message exceeded maximum delivery attempts, routing to DLQ")

		dlqSubject := strings.Replace(msg.Subject, "ts.community.", "ts.dlq.community.", 1)

		// Publish to DLQ
		dlqMsg := natsclient.NewMsg(dlqSubject)
		dlqMsg.Data = msg.Data
		dlqMsg.Header = make(natsclient.Header)
		for k, v := range msg.Header {
			dlqMsg.Header[k] = v
		}
		dlqMsg.Header.Set("Nats-Msg-Id", msg.Header.Get("Nats-Msg-Id"))
		dlqMsg.Header.Set("X-Tacito-DLQ-Original-Subject", msg.Subject)
		dlqMsg.Header.Set("X-Tacito-DLQ-Attempts", fmt.Sprintf("%d", meta.NumDelivered))

		if _, err := js.PublishMsg(dlqMsg, natsclient.Context(ctx)); err != nil {
			logger.Error().Err(err).Msg("failed to publish message to DLQ stream, will retry processing")
			_ = msg.Nak()
			return
		}

		logger.Info().Msg("message successfully routed to DLQ stream, acknowledging original message")
		_ = msg.Ack()
		return
	}

	// Standard processing
	if err := s.handleEvent(ctx, logger, msg); err != nil {
		logger.Error().Err(err).Msg("failed to process message, negative-acknowledging")
		_ = msg.Nak()
		return
	}

	// Success, acknowledge
	_ = msg.Ack()
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

