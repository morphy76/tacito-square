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
