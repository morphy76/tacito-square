package model

import (
	"fmt"
	"time"
	"unicode"
)

// EchoRequest is the payload sent from the Keeper to an agent via NATS request-reply.
type EchoRequest struct {
	Message     string `json:"message"`
	CommunityID string `json:"community_id"`
	TenantID    string `json:"tenant_id"`
}

// EchoReply is the decorated response an agent sends back to the Keeper.
type EchoReply struct {
	AgentName string `json:"agent_name"`
	Decorated string `json:"decorated"`
	Timestamp string `json:"timestamp"` // RFC3339
}

// AgentEchoResult holds the outcome of a single per-agent echo request.
type AgentEchoResult struct {
	AgentName string `json:"agent_name"`
	Decorated string `json:"decorated"`
	Error     string `json:"error,omitempty"`
}

// CommunityEchoResponse is the HTTP response payload of the echo endpoint.
type CommunityEchoResponse struct {
	CommunityID   string            `json:"community_id"`
	WokeCommunity bool              `json:"woke_community"`
	Results       []AgentEchoResult `json:"results,omitempty"`
}

// SanitizeMessage strips non-printable Unicode characters from s and truncates to 1000 characters.
// Returns the sanitized string; the caller must check for an empty result.
func SanitizeMessage(s string) string {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsPrint(r) {
			runes = append(runes, r)
		}
	}
	if len(runes) > 1000 {
		runes = runes[:1000]
	}
	return string(runes)
}

// DecorateMessage wraps a sanitized message with the agent's identity and a timestamp.
func DecorateMessage(agentName, message string) string {
	return fmt.Sprintf("[agent:%s at %s] %s", agentName, time.Now().UTC().Format(time.RFC3339), message)
}
