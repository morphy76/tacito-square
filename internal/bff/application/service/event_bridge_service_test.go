package service_test

import (
	"context"
	"testing"

	"github.com/morphy76/tacito-square/internal/bff/application/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBackendEventClient struct {
	mock.Mock
}

func (m *mockBackendEventClient) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan []byte), args.Error(1)
}

func TestEventBridgeService_StreamEvents_ForwardsChannel(t *testing.T) {
	client := &mockBackendEventClient{}
	svc := service.NewEventBridgeService(client)

	ctx := context.Background()
	tenantID := "tenant-1"
	ch := make(chan []byte)

	client.On("StreamEvents", ctx, tenantID).Return((<-chan []byte)(ch), nil)

	resCh, err := svc.StreamEvents(ctx, tenantID)
	assert.NoError(t, err)
	assert.Equal(t, (<-chan []byte)(ch), resCh)
	mock.AssertExpectationsForObjects(t, client)
}

func TestEventBridgeService_StreamEvents_ContextCancellation(t *testing.T) {
	client := &mockBackendEventClient{}
	svc := service.NewEventBridgeService(client)

	ctx, cancel := context.WithCancel(context.Background())
	tenantID := "tenant-1"
	ch := make(chan []byte, 2)
	ch <- []byte("event-1")

	client.On("StreamEvents", ctx, tenantID).Return((<-chan []byte)(ch), nil)

	resCh, err := svc.StreamEvents(ctx, tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, resCh)

	// Read one event
	evt := <-resCh
	assert.Equal(t, []byte("event-1"), evt)

	// Cancel context and close the mock client channel (simulating client behavior on cancellation)
	cancel()
	close(ch)

	// Assert the returned channel is closed
	_, open := <-resCh
	assert.False(t, open)

	mock.AssertExpectationsForObjects(t, client)
}
