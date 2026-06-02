package qdrant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQdrantLTMAdapter_URLParsing(t *testing.T) {
	// Assert that a URL with http:// prefix is accepted and handled.
	// Since port 9999 is closed, it should fail with a TCP connection error/refused,
	// but the dial target should be successfully resolved (no gRPC protocol/resolver errors).
	_, err := NewQdrantLTMAdapter("http://127.0.0.1:9999", "test_collection", 1536)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to qdrant gRPC")
}
