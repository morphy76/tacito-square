package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/application/ports/inbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// EchoSubscriber listens for EchoRequest messages on the agent's community subject
// and replies with a decorated EchoReply containing reasoning completions.
type EchoSubscriber struct {
	nc          *natsclient.Conn
	agentName   string
	communityID string
	tenantID    string
	processor   inbound.MessageProcessor
	logger      zerolog.Logger
	sub         *natsclient.Subscription
}

// NewEchoSubscriber constructs a new EchoSubscriber. Call Start() to begin listening.
func NewEchoSubscriber(nc *natsclient.Conn, agentName, communityID, tenantID string, processor inbound.MessageProcessor, logger zerolog.Logger) *EchoSubscriber {
	return &EchoSubscriber{
		nc:          nc,
		agentName:   agentName,
		communityID: communityID,
		tenantID:    tenantID,
		processor:   processor,
		logger:      logger,
	}
}

// Start subscribes to the agent's echo subject. Returns an error if subscription fails.
func (s *EchoSubscriber) Start(_ context.Context) error {
	subject := fmt.Sprintf(echoSubjectFormat, s.communityID, s.agentName)
	sub, err := s.nc.Subscribe(subject,
		observability.WrapNATSHandler("nats.echo_handler", s.logger, s.handleEcho))
	if err != nil {
		return fmt.Errorf("echo subscriber: subscribe to %s: %w", subject, err)
	}
	s.sub = sub
	s.logger.Info().Str("subject", subject).Msg("echo subscriber started")
	return nil
}

// Stop drains and unsubscribes.
func (s *EchoSubscriber) Stop() error {
	if s.sub != nil {
		err := s.sub.Drain()
		s.sub = nil
		return err
	}
	return nil
}

func (s *EchoSubscriber) handleEcho(ctx context.Context, logger zerolog.Logger, msg *natsclient.Msg) error {
	var req model.EchoRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		logger.Warn().Err(err).Msg("echo subscriber: malformed payload, ignoring")
		return nil // intentional: malformed messages are silently ignored, no reply sent
	}

	sanitized := model.SanitizeMessage(req.Message)

	logger.Info().
		Str("agent_name", s.agentName).
		Str("community_id", s.communityID).
		Str("tenant_id", req.TenantID).
		Str("message", sanitized).
		Msg("echo request received")

	// Trigger the message processing framework pipeline (Brain reasoning engine)
	brainResult, err := s.processor.ProcessIncomingMessage(ctx, req.Message)
	if err != nil {
		logger.Error().Err(err).Msg("echo subscriber: message processing failed")
		return fmt.Errorf("process incoming message: %w", err)
	}

	decorated := model.DecorateMessage(s.agentName, brainResult)
	now := time.Now().UTC()

	reply := model.EchoReply{
		AgentName: s.agentName,
		Decorated: decorated,
		Timestamp: now.Format(time.RFC3339),
	}

	data, err := json.Marshal(reply)
	if err != nil {
		logger.Error().Err(err).Msg("echo subscriber: failed to marshal reply")
		return fmt.Errorf("marshal echo reply: %w", err)
	}

	if err := msg.Respond(data); err != nil {
		logger.Error().Err(err).Msg("echo subscriber: failed to send reply")
		return fmt.Errorf("respond to echo request: %w", err)
	}

	return nil
}
