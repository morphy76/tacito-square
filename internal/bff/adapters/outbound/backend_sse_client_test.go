package outbound_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/adapters/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendSSEClient_StreamEvents_Success(t *testing.T) {
	events := []string{"event-1", "event-2", "event-3"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/events/stream", r.URL.Path)
		assert.Equal(t, "tenant-a", r.Header.Get("X-Tenant-ID"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok, "expected http.Flusher")

		for _, evt := range events {
			fmt.Fprintf(w, "data: %s\n\n", evt)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	cfg := outbound.BackendSSEClientConfig{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	}
	client := outbound.NewBackendSSEClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ch, err := client.StreamEvents(ctx, "tenant-a")
	require.NoError(t, err)

	received := []string{}
	for msg := range ch {
		received = append(received, string(msg))
		if len(received) == len(events) {
			break
		}
	}

	assert.Len(t, received, 3)
	assert.Contains(t, received[0], "event-1")
	assert.Contains(t, received[1], "event-2")
	assert.Contains(t, received[2], "event-3")
}

func TestBackendSSEClient_ContextCancellation(t *testing.T) {
	started := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		close(started)

		// Keep streaming until client disconnects
		for i := 0; ; i++ {
			fmt.Fprintf(w, "data: event-%d\n\n", i)
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
			if r.Context().Err() != nil {
				return
			}
		}
	}))
	defer srv.Close()

	cfg := outbound.BackendSSEClientConfig{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	}
	client := outbound.NewBackendSSEClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := client.StreamEvents(ctx, "tenant-a")
	require.NoError(t, err)

	// Wait for stream to start
	<-started

	// Read one event
	<-ch

	// Cancel and assert channel closes
	cancel()

	select {
	case _, open := <-ch:
		assert.False(t, open, "channel should be closed after context cancellation")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel to close")
	}
}
