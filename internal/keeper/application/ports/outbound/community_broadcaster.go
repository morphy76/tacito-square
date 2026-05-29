package outbound

import (
	"context"

	"github.com/morphy76/tacito-square/internal/keeper/domain/model"
)

// CommunityBroadcaster is the driven outbound port for per-agent NATS request-reply.
// The subject format used by implementations is: ts.community.{communityID}.agent.{agentName}
type CommunityBroadcaster interface {
	// RequestEcho sends an EchoRequest to the given agent's NATS subject and
	// blocks until it receives an EchoReply or the context deadline is exceeded.
	// Returns an error on timeout or marshal/unmarshal failure.
	RequestEcho(ctx context.Context, communityID, agentName string, req model.EchoRequest) (*model.EchoReply, error)

	// Available reports whether the underlying NATS connection is established.
	// The application service MUST check this before fanning out requests.
	Available() bool
}
