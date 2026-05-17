package cache

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestCacheAdapter_SetGet_RoundTrip(t *testing.T) {
	mock := newMockRedis()
	adapter := NewCacheAdapter(mock, "keeper")

	input := testStruct{Name: "prompt-123", Value: 42}
	err := adapter.Set(context.Background(), "my-prompt", input, time.Hour)
	require.NoError(t, err)

	var result testStruct
	err = adapter.Get(context.Background(), "my-prompt", &result)
	require.NoError(t, err)
	assert.Equal(t, "prompt-123", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestCacheAdapter_Get_Miss(t *testing.T) {
	mock := newMockRedis()
	adapter := NewCacheAdapter(mock, "keeper")

	var result testStruct
	err := adapter.Get(context.Background(), "nonexistent", &result)

	require.ErrorIs(t, err, outbound.ErrCacheMiss)
}

func TestCacheAdapter_Invalidate(t *testing.T) {
	mock := newMockRedis()
	adapter := NewCacheAdapter(mock, "keeper")

	input := testStruct{Name: "to-invalidate", Value: 1}
	require.NoError(t, adapter.Set(context.Background(), "inv-key", input, time.Hour))

	err := adapter.Invalidate(context.Background(), "inv-key")
	require.NoError(t, err)

	var result testStruct
	err = adapter.Get(context.Background(), "inv-key", &result)
	require.ErrorIs(t, err, outbound.ErrCacheMiss)
}

func TestCacheAdapter_KeyNamespacing(t *testing.T) {
	mock := newMockRedis()
	adapter := NewCacheAdapter(mock, "keeper")

	require.NoError(t, adapter.Set(context.Background(), "foo", "bar", time.Hour))

	// Verify the internal key is namespaced
	expectedKey := "ts:keeper:cache:foo"
	_, exists := mock.store[expectedKey]
	assert.True(t, exists, "key should be namespaced as %s", expectedKey)
}

func TestCacheAdapter_TTL(t *testing.T) {
	mock := newMockRedis()
	adapter := NewCacheAdapter(mock, "agent")

	ttl := 30 * time.Minute
	require.NoError(t, adapter.Set(context.Background(), "ttl-key", "value", ttl))

	expectedKey := "ts:agent:cache:ttl-key"
	assert.Equal(t, ttl, mock.ttls[expectedKey], "TTL should be set on the cached key")
}

func TestCacheAdapter_ComponentIsolation(t *testing.T) {
	mock := newMockRedis()
	keeperCache := NewCacheAdapter(mock, "keeper")
	agentCache := NewCacheAdapter(mock, "agent")

	require.NoError(t, keeperCache.Set(context.Background(), "shared-key", "keeper-val", time.Hour))
	require.NoError(t, agentCache.Set(context.Background(), "shared-key", "agent-val", time.Hour))

	var keeperResult string
	require.NoError(t, keeperCache.Get(context.Background(), "shared-key", &keeperResult))
	assert.Equal(t, "keeper-val", keeperResult)

	var agentResult string
	require.NoError(t, agentCache.Get(context.Background(), "shared-key", &agentResult))
	assert.Equal(t, "agent-val", agentResult)
}
