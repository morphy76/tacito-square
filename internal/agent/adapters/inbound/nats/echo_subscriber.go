package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	natsclient "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// EchoSubscriber listens for EchoRequest messages on the agent's community subject
// and replies with a decorated EchoReply.
type EchoSubscriber struct {
	nc          *natsclient.Conn
	agentName   string
	communityID string
	tenantID    string
	logger      zerolog.Logger
	sub         *natsclient.Subscription
}

// NewEchoSubscriber constructs a new EchoSubscriber. Call Start() to begin listening.
func NewEchoSubscriber(nc *natsclient.Conn, agentName, communityID, tenantID string, logger zerolog.Logger) *EchoSubscriber {
	return &EchoSubscriber{
		nc:          nc,
		agentName:   agentName,
		communityID: communityID,
		tenantID:    tenantID,
		logger:      logger,
	}
}

// Start subscribes to the agent's echo subject. Returns an error if subscription fails.
func (s *EchoSubscriber) Start(_ context.Context) error {
	subject := fmt.Sprintf(echoSubjectFormat, s.communityID, s.agentName)
	sub, err := s.nc.Subscribe(subject, s.handleEcho)
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

func (s *EchoSubscriber) handleEcho(msg *natsclient.Msg) {
	var req model.EchoRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		s.logger.Warn().Err(err).Msg("echo subscriber: malformed payload, ignoring")
		return
	}

	sanitized := model.SanitizeMessage(req.Message)

	s.logger.Info().
		Str("agent_name", s.agentName).
		Str("community_id", s.communityID).
		Str("tenant_id", req.TenantID).
		Str("message", sanitized).
		Msg("echo request received")

	decorated := model.DecorateMessage(s.agentName, sanitized)
	now := time.Now().UTC()

	reply := model.EchoReply{
		AgentName: s.agentName,
		Decorated: decorated,
		Timestamp: now.Format(time.RFC3339),
	}

	data, err := json.Marshal(reply)
	if err != nil {
		s.logger.Error().Err(err).Msg("echo subscriber: failed to marshal reply")
		return
	}

	if err := msg.Respond(data); err != nil {
		s.logger.Error().Err(err).Msg("echo subscriber: failed to send reply")
	}
}
