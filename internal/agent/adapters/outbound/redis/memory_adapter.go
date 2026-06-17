package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/morphy76/tacito-square/internal/shared/observability"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// RedisMemoryAdapter implements the outbound ShortTermMemory interface using Redis.
type RedisMemoryAdapter struct {
	client *redis.Client
	ttl    time.Duration
	tracer trace.Tracer
}

// NewRedisMemoryAdapter creates a new RedisMemoryAdapter instance.
func NewRedisMemoryAdapter(redisURL string, ttl time.Duration) (*RedisMemoryAdapter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	// Ping to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisMemoryAdapter{
		client: client,
		ttl:    ttl,
		tracer: otel.Tracer("redis"),
	}, nil
}

// Close closes the underlying Redis client connection pool.
func (a *RedisMemoryAdapter) Close() error {
	return a.client.Close()
}

// Ping verifies Redis connectivity for health probes.
func (a *RedisMemoryAdapter) Ping(ctx context.Context) error {
	return a.client.Ping(ctx).Err()
}

// formatKey returns the strictly multi-tenant isolated key structure.
func (a *RedisMemoryAdapter) formatKey(tenantID, agentID, threadID string) string {
	return fmt.Sprintf("ts:tenant:%s:agent:%s:stm:%s", tenantID, agentID, threadID)
}

// Append adds a new memory entry to the specified thread, updating the TTL.
func (a *RedisMemoryAdapter) Append(ctx context.Context, tenantID, agentID, threadID string, entry model.MemoryEntry) error {
	start := time.Now()

	// 1. Start sub-span for Redis Append
	ctx, span := a.tracer.Start(ctx, "redis.append",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "rpush"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	// Validate entry before serialization
	if err := entry.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Msg("invalid memory entry validation failure")
		return fmt.Errorf("invalid memory entry: %w", err)
	}

	key := a.formatKey(tenantID, agentID, threadID)
	data, err := json.Marshal(entry)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to marshal memory entry to JSON")
		return fmt.Errorf("failed to marshal memory entry: %w", err)
	}

	// 2. Perform pipeline: Append then Expire
	pipe := a.client.Pipeline()
	pipe.RPush(ctx, key, string(data))
	pipe.Expire(ctx, key, a.ttl)

	_, err = pipe.Exec(ctx)
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to append short-term memory entry in Redis")
	} else {
		span.SetStatus(codes.Ok, "")
		logger.Debug().
			Str("key", key).
			Str("entry_role", entry.Role).
			Msg("short-term memory entry appended to Redis list successfully")
	}

	// 3. Record outbound latency metrics
	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "redis"),
			attribute.String("operation", "append"),
			attribute.String("status", status),
		),
	)

	// Record agent stm metrics
	stmAttrs := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "write"),
	)
	stmAttrsWithStatus := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "write"),
		attribute.String("status", status),
	)
	observability.AgentSTMOperationsTotal.Add(ctx, 1, stmAttrsWithStatus)
	observability.AgentSTMOperationDuration.Record(ctx, duration, stmAttrs)

	if err != nil {
		return fmt.Errorf("redis pipe exec failed: %w", err)
	}
	return nil
}

// Get retrieves the sliding window of conversational history up to a limit.
func (a *RedisMemoryAdapter) Get(ctx context.Context, tenantID, agentID, threadID string, limit int) ([]model.MemoryEntry, error) {
	start := time.Now()

	// 1. Start sub-span for Redis Get
	ctx, span := a.tracer.Start(ctx, "redis.get",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "lrange"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
			attribute.Int("limit", limit),
		),
	)
	defer span.End()

	key := a.formatKey(tenantID, agentID, threadID)

	// Since LRange retrieves by index bounds, a limit of N means retrieving the last N elements.
	// Redis lists are 0-indexed. Getting the last N: Start index = -N, End index = -1.
	startIdx := int64(-limit)
	if limit <= 0 {
		startIdx = 0 // get all if limit <= 0
	}

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	items, err := a.client.LRange(ctx, key, startIdx, -1).Result()
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to retrieve short-term memory from Redis")
	} else {
		span.SetStatus(codes.Ok, "")
		logger.Debug().
			Str("key", key).
			Int("retrieved_count", len(items)).
			Msg("short-term memory history retrieved from Redis successfully")
	}

	// 2. Record metrics
	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "redis"),
			attribute.String("operation", "get"),
			attribute.String("status", status),
		),
	)

	// Record agent stm metrics
	stmAttrs := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "read"),
	)
	stmAttrsWithStatus := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "read"),
		attribute.String("status", status),
	)
	observability.AgentSTMOperationsTotal.Add(ctx, 1, stmAttrsWithStatus)
	observability.AgentSTMOperationDuration.Record(ctx, duration, stmAttrs)

	if err != nil {
		return nil, fmt.Errorf("redis lrange failed: %w", err)
	}

	entries := make([]model.MemoryEntry, 0, len(items))
	for _, item := range items {
		var entry model.MemoryEntry
		if err := json.Unmarshal([]byte(item), &entry); err != nil {
			// Skip corrupted or unparseable entries rather than failing the whole history
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Clear deletes all entries associated with the specified thread.
func (a *RedisMemoryAdapter) Clear(ctx context.Context, tenantID, agentID, threadID string) error {
	start := time.Now()

	// 1. Start sub-span for Redis Clear
	ctx, span := a.tracer.Start(ctx, "redis.clear",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "del"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatKey(tenantID, agentID, threadID)
	err := a.client.Del(ctx, key).Err()
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "failure"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to clear short-term memory from Redis")
	} else {
		span.SetStatus(codes.Ok, "")
		logger.Debug().
			Str("key", key).
			Msg("short-term memory history cleared from Redis successfully")
	}

	// 2. Record metrics
	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "redis"),
			attribute.String("operation", "clear"),
			attribute.String("status", status),
		),
	)

	// Record agent stm metrics
	stmAttrs := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "delete"),
	)
	stmAttrsWithStatus := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "delete"),
		attribute.String("status", status),
	)
	observability.AgentSTMOperationsTotal.Add(ctx, 1, stmAttrsWithStatus)
	observability.AgentSTMOperationDuration.Record(ctx, duration, stmAttrs)

	if err != nil {
		return fmt.Errorf("redis del failed: %w", err)
	}
	return nil
}

// RollbackLast pops the last entry in the short-term memory list for the thread.
func (a *RedisMemoryAdapter) RollbackLast(ctx context.Context, tenantID, agentID, threadID string) error {
	start := time.Now()

	ctx, span := a.tracer.Start(ctx, "redis.rollback_last",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "rpop"),
			attribute.String("tenant_id", tenantID),
			attribute.String("agent_id", agentID),
			attribute.String("thread_id", threadID),
		),
	)
	defer span.End()

	logger := zerolog.Ctx(ctx).With().
		Str("db.system", "redis").
		Str("tenant_id", tenantID).
		Str("agent_id", agentID).
		Str("thread_id", threadID).
		Logger()

	key := a.formatKey(tenantID, agentID, threadID)
	err := a.client.RPop(ctx, key).Err()
	duration := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		if err == redis.Nil {
			// List is empty or key doesn't exist, handle gracefully
			err = nil
		} else {
			status = "failure"
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Error().Err(err).Str("key", key).Msg("failed to pop last entry from Redis list")
		}
	}

	if err == nil {
		span.SetStatus(codes.Ok, "")
		logger.Debug().
			Str("key", key).
			Msg("short-term memory last entry rolled back successfully")
	}

	// Record outbound latency metrics
	observability.OutboundDependencyDuration.Record(ctx, duration,
		otelmetric.WithAttributes(
			attribute.String("dependency", "redis"),
			attribute.String("operation", "rollback_last"),
			attribute.String("status", status),
		),
	)

	// Record agent stm metrics
	stmAttrs := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "rollback_last"),
	)
	stmAttrsWithStatus := otelmetric.WithAttributes(
		attribute.String("agent", agentID),
		attribute.String("operation", "rollback_last"),
		attribute.String("status", status),
	)
	observability.AgentSTMOperationsTotal.Add(ctx, 1, stmAttrsWithStatus)
	observability.AgentSTMOperationDuration.Record(ctx, duration, stmAttrs)

	if err != nil {
		return fmt.Errorf("redis rpop failed: %w", err)
	}
	return nil
}

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
func (a *RedisMemoryAdapter) Lock(ctx context.Context, tenantID, threadID string) (bool, error) {
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
	ttl := 30 * time.Second

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			span.RecordError(ctx.Err())
			span.SetStatus(codes.Error, ctx.Err().Error())
			return false, ctx.Err()
		case <-timeout:
			err := fmt.Errorf("lock acquisition timed out for thread %s", threadID)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logger.Warn().Msg("lock acquisition timed out")
			return false, err
		case <-ticker.C:
			success, err := a.client.SetNX(ctx, key, "locked", ttl).Result()
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return false, fmt.Errorf("failed to execute SetNX: %w", err)
			}
			if success {
				span.SetStatus(codes.Ok, "")
				logger.Debug().Str("key", key).Msg("acquired thread lock successfully")
				return true, nil
			}
		}
	}
}

// Unlock releases the thread lock.
func (a *RedisMemoryAdapter) Unlock(ctx context.Context, tenantID, threadID string) error {
	ctx, span := a.tracer.Start(ctx, "redis.unlock",
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

	key := a.formatLockKey(tenantID, threadID)
	err := a.client.Del(ctx, key).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error().Err(err).Str("key", key).Msg("failed to release thread lock in Redis")
		return fmt.Errorf("failed to release thread lock: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logger.Debug().Str("key", key).Msg("released thread lock successfully")
	return nil
}
