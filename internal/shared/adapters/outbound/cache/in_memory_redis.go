package cache

import (
	"context"
	"sync"
	"time"

	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
)

// InMemoryRedis is a thread-safe, in-memory implementation of RedisCache
// that acts as a resilient fallback when Redis is unconfigured or offline.
type InMemoryRedis struct {
	mu    sync.RWMutex
	store map[string][]byte
	ttls  map[string]time.Time
}

// NewInMemoryRedis creates a new InMemoryRedis instance and starts its cleanup loop.
func NewInMemoryRedis() *InMemoryRedis {
	r := &InMemoryRedis{
		store: make(map[string][]byte),
		ttls:  make(map[string]time.Time),
	}
	go r.cleanupLoop()
	return r
}

// Set stores a key-value pair with a TTL.
func (r *InMemoryRedis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store[key] = value
	if ttl > 0 {
		r.ttls[key] = time.Now().Add(ttl)
	} else {
		delete(r.ttls, key)
	}
	return nil
}

// Get retrieves a key's value. Returns ErrCacheMiss if key does not exist or has expired.
func (r *InMemoryRedis) Get(ctx context.Context, key string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check expiration
	if expiry, exists := r.ttls[key]; exists && time.Now().After(expiry) {
		return nil, outbound.ErrCacheMiss
	}

	data, ok := r.store[key]
	if !ok {
		return nil, outbound.ErrCacheMiss
	}
	return data, nil
}

// Del removes a key.
func (r *InMemoryRedis) Del(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.store, key)
	delete(r.ttls, key)
	return nil
}

func (r *InMemoryRedis) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, expiry := range r.ttls {
			if now.After(expiry) {
				delete(r.store, key)
				delete(r.ttls, key)
			}
		}
		r.mu.Unlock()
	}
}
