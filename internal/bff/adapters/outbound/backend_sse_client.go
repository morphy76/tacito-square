package outbound

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Compile-time interface satisfaction assertion.
var _ outbound.BackendEventClient = (*BackendSSEClient)(nil)

// BackendSSEClientConfig holds configurable parameters for the backend SSE outbound adapter.
type BackendSSEClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// BackendSSEClient is a driven adapter implementing outbound.BackendEventClient,
// establishing an SSE HTTP stream from the backend event source.
type BackendSSEClient struct {
	cfg    BackendSSEClientConfig
	client *http.Client
}

// NewBackendSSEClient constructs a new BackendSSEClient with OTel-instrumented transport.
func NewBackendSSEClient(cfg BackendSSEClientConfig) *BackendSSEClient {
	// SSE streams are long-lived; use no timeout on the HTTP client itself.
	// Deadline is enforced via the passed context.Context.
	return &BackendSSEClient{
		cfg: cfg,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// StreamEvents establishes an SSE stream with the backend and returns a read-only channel
// emitting raw event byte payloads. The channel is closed when the context is cancelled
// or the backend closes the connection.
func (s *BackendSSEClient) StreamEvents(ctx context.Context, tenantID string) (<-chan []byte, error) {
	endpoint := fmt.Sprintf("%s/api/v1/events/stream", s.cfg.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("backend sse: build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend sse: connect: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("backend sse: server returned %d", resp.StatusCode)
	}

	ch := make(chan []byte)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if ctx.Err() != nil {
				log.Debug().Str("tenant_id", tenantID).Msg("SSE stream closed due to context cancellation")
				return
			}
			if strings.HasPrefix(line, "data:") {
				payload := strings.TrimPrefix(line, "data:")
				payload = strings.TrimSpace(payload)
				select {
				case ch <- []byte(payload):
				case <-ctx.Done():
					log.Debug().Str("tenant_id", tenantID).Msg("SSE stream closed due to context done on send")
					return
				}
			}
		}

		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			log.Warn().Err(err).Str("tenant_id", tenantID).Msg("SSE scanner error")
		}
	}()

	return ch, nil
}
