package outbound_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/bff/adapters/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeeperClient_Ping_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := outbound.KeeperClientConfig{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	}
	client := outbound.NewKeeperHTTPClient(cfg)

	err := client.Ping(context.Background())
	require.NoError(t, err)
}

func TestKeeperClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client disconnects (context cancelled by timeout).
		<-r.Context().Done()
	}))
	// Close client connections first so srv.Close() does not block.
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })

	cfg := outbound.KeeperClientConfig{
		BaseURL: srv.URL,
		Timeout: 100 * time.Millisecond,
	}
	client := outbound.NewKeeperHTTPClient(cfg)

	err := client.Ping(context.Background())
	assert.Error(t, err, "expected timeout error")
}

func TestKeeperClient_CompileTimeAssertion(t *testing.T) {
	// Compile-time interface check covered by var _ below the struct definition.
}
