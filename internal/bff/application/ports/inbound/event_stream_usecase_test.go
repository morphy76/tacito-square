package inbound_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/inbound"
)

type mockEventStreamUseCase struct{}

func (m *mockEventStreamUseCase) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	return nil, nil
}

// Compile-time assertion
var _ inbound.EventStreamUseCase = (*mockEventStreamUseCase)(nil)

func TestEventStreamUseCaseInterface(t *testing.T) {
	// Assertions are checked at compile time
}
