package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// formatStateKey returns the Redis key for a thread's orchestration state.
func (a *RedisMemoryAdapter) formatStateKey(tenantID, threadID string) string {
	return fmt.Sprintf("ts:tenant:%s:state:%s", tenantID, threadID)
}

// formatLockKey returns the Redis key for a thread lock.
func (a *RedisMemoryAdapter) formatLockKey(tenantID, threadID string) string {
	return fmt.Sprintf("ts:tenant:%s:lock:%s", tenantID, threadID)
}

// SaveState serializes and persists the thread's orchestration state in Redis.
func (a *RedisMemoryAdapter) SaveState(ctx context.Context, tenantID, threadID string, state model.OrchestrationState) error {
	ctx, span := a.tracer.Start(ctx, "redis.save_state",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "set"),
			attribute.String("tenant_id", tenantID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("thread_id", threadID).
		Logger()

	if err := state.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("invalid orchestration state: %w", err)
	}

	key := a.formatStateKey(tenantID, threadID)
	data, err := json.Marshal(state)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to marshal orchestration state: %w", err)
	}

	err = a.client.Set(ctx, key, string(data), a.ttl).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to save orchestration state in Redis")
		return fmt.Errorf("failed to set state key: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Str("key", key).Msg("saved orchestration state in Redis successfully")
	return nil
}

// GetState retrieves and deserializes the thread's orchestration state from Redis.
func (a *RedisMemoryAdapter) GetState(ctx context.Context, tenantID, threadID string) (*model.OrchestrationState, error) {
	ctx, span := a.tracer.Start(ctx, "redis.get_state",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "get"),
			attribute.String("tenant_id", tenantID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatStateKey(tenantID, threadID)
	val, err := a.client.Get(ctx, key).Result()
	if err == redis.Nil {
		span.SetStatus(codes.Ok, "")
		return nil, nil // Not found is a valid scenario
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to get orchestration state from Redis")
		return nil, fmt.Errorf("failed to get state key: %w", err)
	}

	var state model.OrchestrationState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to unmarshal orchestration state: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Str("key", key).Msg("retrieved orchestration state from Redis successfully")
	return &state, nil
}

// ClearState deletes the thread's orchestration state from Redis.
func (a *RedisMemoryAdapter) ClearState(ctx context.Context, tenantID, threadID string) error {
	ctx, span := a.tracer.Start(ctx, "redis.clear_state",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "del"),
			attribute.String("tenant_id", tenantID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatStateKey(tenantID, threadID)
	err := a.client.Del(ctx, key).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to delete orchestration state in Redis")
		return fmt.Errorf("failed to delete state key: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Str("key", key).Msg("cleared orchestration state in Redis successfully")
	return nil
}

// Lock attempts to acquire a distributed lock on the thread.
func (a *RedisMemoryAdapter) Lock(ctx context.Context, tenantID, threadID string) (string, bool, error) {
	ctx, span := a.tracer.Start(ctx, "redis.lock",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "setnx"),
			attribute.String("tenant_id", tenantID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatLockKey(tenantID, threadID)
	token := uuid.New().String()

	// Try acquiring the lock immediately first
	success, err := a.client.SetNX(ctx, key, token, a.lockTTL).Result()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", false, fmt.Errorf("failed to execute SetNX: %w", err)
	}
	if success {
		span.SetStatus(codes.Ok, "")
		logger.Debug().Str("key", key).Msg("acquired thread lock successfully")
		return token, true, nil
	}

	timeout := time.After(a.lockTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, ctx.Err().Error())
			return "", false, ctx.Err()
		case <-timeout:
			err := fmt.Errorf("lock acquisition timed out for thread %s", threadID)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Warn().Msg("lock acquisition timed out")
			return "", false, err
		case <-ticker.C:
			success, err := a.client.SetNX(ctx, key, token, a.lockTTL).Result()
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return "", false, fmt.Errorf("failed to execute SetNX: %w", err)
			}
			if success {
				span.SetStatus(codes.Ok, "")
				logger.Debug().Str("key", key).Msg("acquired thread lock successfully")
				return token, true, nil
			}
		}
	}
}

var unlockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`)

// Unlock releases the thread lock if owned by the caller.
func (a *RedisMemoryAdapter) Unlock(ctx context.Context, tenantID, threadID, token string) error {
	ctx, span := a.tracer.Start(ctx, "redis.unlock",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "eval"),
			attribute.String("tenant_id", tenantID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatLockKey(tenantID, threadID)
	res, err := unlockScript.Run(ctx, a.client, []string{key}, token).Int64()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to execute unlock Lua script in Redis")
		return fmt.Errorf("failed to release thread lock: %w", err)
	}

	if res == 0 {
		err := fmt.Errorf("lock release failed: lock not held or owned by this token")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Warn().Str("key", key).Msg("attempted to release lock not owned by caller")
		return err
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Str("key", key).Msg("released thread lock successfully")
	return nil
}
