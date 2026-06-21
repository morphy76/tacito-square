package outbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
)

type mockBackendEventClient struct{}

func (m *mockBackendEventClient) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	return nil, nil
}

// Compile-time assertion
var _ outbound.BackendEventClient = (*mockBackendEventClient)(nil)

func TestBackendEventClientInterface(t *testing.T) {
	// Assertions are checked at compile time
}
