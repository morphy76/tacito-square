package outbound

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/application/ports/outbound"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Compile-time interface satisfaction assertion.
var _ outbound.KeeperClient = (*KeeperHTTPClient)(nil)

// KeeperClientConfig holds configurable parameters for the Keeper outbound adapter.
type KeeperClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// KeeperHTTPClient is a driven adapter implementing outbound.KeeperClient,
// issuing HTTP calls to the Keeper core service with OTel trace propagation.
type KeeperHTTPClient struct {
	cfg    KeeperClientConfig
	client *http.Client
}

// NewKeeperHTTPClient constructs a new KeeperHTTPClient with an OTel-instrumented HTTP transport.
func NewKeeperHTTPClient(cfg KeeperClientConfig) *KeeperHTTPClient {
	return &KeeperHTTPClient{
		cfg: cfg,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   cfg.Timeout,
		},
	}
}

// Ping verifies connectivity to the Keeper service by calling its /healthz endpoint.
func (k *KeeperHTTPClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, k.cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.cfg.BaseURL+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("keeper client: build ping request: %w", err)
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return fmt.Errorf("keeper client: ping call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("keeper client: /healthz returned %d", resp.StatusCode)
	}

	return nil
}
