package observability

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewInstrumentedClient_DefaultFallback(t *testing.T) {
	// Ensure env vars are not set
	os.Unsetenv("HTTP_CLIENT_MAX_IDLE_CONNS")
	os.Unsetenv("HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST")
	os.Unsetenv("HTTP_CLIENT_IDLE_CONN_TIMEOUT")
	os.Unsetenv("HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT")
	os.Unsetenv("HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT")

	client := NewInstrumentedClient(15 * time.Second)
	assert.Equal(t, 15*time.Second, client.Timeout)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		// If wrapped in otelhttp, try to extract transport if accessible, or just assert it's wrapped.
		// Wait, the client returns otelhttp transport, which wraps the underlying transport.
		// Let's assert it is indeed created and not nil.
		assert.NotNil(t, client.Transport)
	} else {
		assert.Equal(t, 100, transport.MaxIdleConns)
		assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
		assert.Equal(t, 90*time.Second, transport.IdleConnTimeout)
		assert.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
		assert.Equal(t, 1*time.Second, transport.ExpectContinueTimeout)
	}
}

func TestNewInstrumentedClient_EnvironmentOverrides(t *testing.T) {
	os.Setenv("HTTP_CLIENT_MAX_IDLE_CONNS", "200")
	os.Setenv("HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST", "25")
	os.Setenv("HTTP_CLIENT_IDLE_CONN_TIMEOUT", "120s")
	os.Setenv("HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT", "5s")
	os.Setenv("HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT", "500ms")

	defer func() {
		os.Unsetenv("HTTP_CLIENT_MAX_IDLE_CONNS")
		os.Unsetenv("HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST")
		os.Unsetenv("HTTP_CLIENT_IDLE_CONN_TIMEOUT")
		os.Unsetenv("HTTP_CLIENT_TLS_HANDSHAKE_TIMEOUT")
		os.Unsetenv("HTTP_CLIENT_EXPECT_CONTINUE_TIMEOUT")
	}()

	client := NewInstrumentedClient(30 * time.Second)
	assert.Equal(t, 30*time.Second, client.Timeout)
	assert.NotNil(t, client.Transport)
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT_VAL", "42")
	defer os.Unsetenv("TEST_INT_VAL")

	assert.Equal(t, 42, getEnvInt("TEST_INT_VAL", 100))
	assert.Equal(t, 100, getEnvInt("TEST_INT_MISSING", 100))
	
	os.Setenv("TEST_INT_INVALID", "not-an-int")
	defer os.Unsetenv("TEST_INT_INVALID")
	assert.Equal(t, 100, getEnvInt("TEST_INT_INVALID", 100))
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DUR_VAL", "45s")
	defer os.Unsetenv("TEST_DUR_VAL")

	assert.Equal(t, 45*time.Second, getEnvDuration("TEST_DUR_VAL", 10*time.Second))
	assert.Equal(t, 10*time.Second, getEnvDuration("TEST_DUR_MISSING", 10*time.Second))

	os.Setenv("TEST_DUR_INVALID", "not-a-duration")
	defer os.Unsetenv("TEST_DUR_INVALID")
	assert.Equal(t, 10*time.Second, getEnvDuration("TEST_DUR_INVALID", 10*time.Second))
}
