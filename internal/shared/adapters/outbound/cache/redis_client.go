package cache

import (
	"context"
	"time"

	"github.com/morphy76/tacito-square/internal/shared/ports/outbound"
	"github.com/redis/go-redis/v9"
)

// RedisClientWrapper implements the RedisCache interface using the go-redis client.
type RedisClientWrapper struct {
	client redis.UniversalClient
}

// NewRedisClientWrapper creates a new RedisClientWrapper.
func NewRedisClientWrapper(client redis.UniversalClient) *RedisClientWrapper {
	return &RedisClientWrapper{client: client}
}

// Set stores a key-value pair with a TTL.
func (w *RedisClientWrapper) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return w.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a key's value, mapping redis.Nil to outbound.ErrCacheMiss.
func (w *RedisClientWrapper) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := w.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, outbound.ErrCacheMiss
		}
		return nil, err
	}
	return data, nil
}

// Del removes a key.
func (w *RedisClientWrapper) Del(ctx context.Context, key string) error {
	return w.client.Del(ctx, key).Err()
}
