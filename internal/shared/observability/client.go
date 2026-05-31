package observability

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewInstrumentedClient returns a thread-safe HTTP client pre-configured with
// connection pooling boundaries and OpenTelemetry tracing propagation.
func NewInstrumentedClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	maxIdleConns := getEnvInt("HTTP_CLIENT_MAX_IDLE_CONNS", 100)
	maxIdleConnsPerHost := getEnvInt("HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST", 10)
	idleConnTimeout := getEnvDuration("HTTP_CLIENT_IDLE_CONN_TIMEOUT", 90*time.Second)
	tlsHandshakeTimeout := getEnvDuration("HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT", 10*time.Second)
	expectContinueTimeout := getEnvDuration("HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT", 1*time.Second)

	transport := &http.Transport{
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(transport),
	}
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
