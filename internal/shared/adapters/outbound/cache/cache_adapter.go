// Package cache implements the Cache port using Redis with component-namespaced keys.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
)

// RedisCache abstracts Redis operations for the cache adapter.
type RedisCache interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
}

// mockRedis is a test double for Redis cache operations.
type mockRedis struct {
	store map[string][]byte
	ttls  map[string]time.Duration
}

func newMockRedis() *mockRedis {
	return &mockRedis{
		store: make(map[string][]byte),
		ttls:  make(map[string]time.Duration),
	}
}

func (m *mockRedis) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	m.store[key] = value
	m.ttls[key] = ttl
	return nil
}

func (m *mockRedis) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := m.store[key]
	if !ok {
		return nil, outbound.ErrCacheMiss
	}
	return data, nil
}

func (m *mockRedis) Del(_ context.Context, key string) error {
	delete(m.store, key)
	return nil
}

// CacheAdapter implements the Cache port using Redis.
type CacheAdapter struct {
	client    RedisCache
	component string // e.g., "keeper", "agent", "bff"
}

// NewCacheAdapter creates a new cache adapter for a specific component.
func NewCacheAdapter(client RedisCache, component string) *CacheAdapter {
	return &CacheAdapter{
		client:    client,
		component: component,
	}
}

// cacheKey returns the namespaced Redis key: ts:{component}:cache:{key}
func (a *CacheAdapter) cacheKey(key string) string {
	return fmt.Sprintf("ts:%s:cache:%s", a.component, key)
}

// Set serializes and stores a value with TTL.
func (a *CacheAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	if err := a.client.Set(ctx, a.cacheKey(key), data, ttl); err != nil {
		return fmt.Errorf("cache set: %w", err)
	}
	return nil
}

// Get retrieves and deserializes a cached value. Returns ErrCacheMiss on miss.
func (a *CacheAdapter) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := a.client.Get(ctx, a.cacheKey(key))
	if err != nil {
		return err // propagate ErrCacheMiss directly
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache unmarshal: %w", err)
	}
	return nil
}

// Invalidate removes a cached entry.
func (a *CacheAdapter) Invalidate(ctx context.Context, key string) error {
	if err := a.client.Del(ctx, a.cacheKey(key)); err != nil {
		return fmt.Errorf("cache invalidate: %w", err)
	}
	return nil
}
