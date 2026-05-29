package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

const echoSubjectFormat = "ts.community.%s.agent.%s"

// NATSCommunityBroadcaster implements outbound.CommunityBroadcaster via NATS request-reply.
type NATSCommunityBroadcaster struct {
	nc     *nats.Conn
	logger zerolog.Logger
}

var _ outbound.CommunityBroadcaster = (*NATSCommunityBroadcaster)(nil)

// NewNATSCommunityBroadcaster creates a new instance of NATSCommunityBroadcaster.
func NewNATSCommunityBroadcaster(nc *nats.Conn, logger zerolog.Logger) *NATSCommunityBroadcaster {
	return &NATSCommunityBroadcaster{
		nc:     nc,
		logger: logger,
	}
}

// Available reports whether the underlying NATS connection is established.
func (b *NATSCommunityBroadcaster) Available() bool {
	return b.nc != nil && b.nc.IsConnected()
}

// RequestEcho sends an EchoRequest to a single agent subject and waits for an EchoReply.
// Subject format: ts.community.{communityID}.agent.{agentName}
func (b *NATSCommunityBroadcaster) RequestEcho(ctx context.Context, communityID, agentName string, req model.EchoRequest) (*model.EchoReply, error) {
	subject := fmt.Sprintf(echoSubjectFormat, communityID, agentName)

	b.logger.Debug().
		Str("subject", subject).
		Str("community_id", communityID).
		Str("agent_name", agentName).
		Msg("Sending NATS echo request")

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal echo request: %w", err)
	}

	msg, err := b.nc.RequestMsgWithContext(ctx, &nats.Msg{
		Subject: subject,
		Data:    payload,
	})
	if err != nil {
		return nil, fmt.Errorf("nats request to %s: %w", subject, err)
	}

	var reply model.EchoReply
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return nil, fmt.Errorf("unmarshal echo reply: %w", err)
	}

	b.logger.Debug().
		Str("subject", subject).
		Str("community_id", communityID).
		Str("agent_name", agentName).
		Msg("Received NATS echo reply")

	return &reply, nil
}
