package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

type mockKeeperClient struct{}

func (m *mockKeeperClient) Ping(ctx context.Context) error {
	return nil
}

// Compile-time assertion
var _ outbound.KeeperClient = (*mockKeeperClient)(nil)

func TestKeeperClientInterface(t *testing.T) {
	// Assertions are checked at compile time
}
