package outbound_test

import (
	"testing"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

// TestCommunityBroadcasterIsInterface asserts the interface is reachable.
// Real behaviour is tested in the adapter and service layers.
func TestCommunityBroadcasterIsInterface(t *testing.T) {
	var _ outbound.CommunityBroadcaster = (outbound.CommunityBroadcaster)(nil)
}
