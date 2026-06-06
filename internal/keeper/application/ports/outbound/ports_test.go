package outbound_test

import (
	"testing"

	"github.com/morphy76/tacito-square/internal/keeper/application/ports/outbound"
)

func TestKeeperPortsReachable(t *testing.T) {
	// Compile-time interface checks (will fail to compile in RED phase)
	var _ outbound.EventPublisher = (outbound.EventPublisher)(nil)
	var _ outbound.EventSubscriber = (outbound.EventSubscriber)(nil)
}
