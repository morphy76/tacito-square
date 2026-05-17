// Package outbound defines shared outbound port interfaces used across components.
package outbound

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when a cache lookup finds no entry.
var ErrCacheMiss = errors.New("cache miss")

// Cache is the outbound port for infrastructure caching (Redis-backed).
type Cache interface {
	// Get retrieves a cached value and unmarshals it into dest.
	Get(ctx context.Context, key string, dest interface{}) error
	// Set stores a value with a TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Invalidate removes a cached entry.
	Invalidate(ctx context.Context, key string) error
}
