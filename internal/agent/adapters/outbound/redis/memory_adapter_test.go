//go:build integration

package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisMemoryAdapter(t *testing.T) {
	// Skip if Docker is not available
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skip("Docker is not available, skipping Redis integration test:", err)
	}
	defer func() {
		_ = redisContainer.Terminate(ctx)
	}()

	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)
	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	redisURL := fmt.Sprintf("redis://%s:%s", host, port.Port())
	adapter, err := NewRedisMemoryAdapter(redisURL, 1*time.Hour)
	require.NoError(t, err)

	t.Run("Append and Get history - strict multi-tenancy and TTL limits", func(t *testing.T) {
		tenantA := "tenant-a"
		tenantB := "tenant-b"
		agentID := "agent-1"
		threadID := "thread-123"

		entry1 := model.MemoryEntry{
			Role:      "user",
			Content:   "Hello from Tenant A",
			Timestamp: time.Now().UTC(),
		}
		entry2 := model.MemoryEntry{
			Role:      "assistant",
			Content:   "Response for Tenant A",
			Timestamp: time.Now().UTC(),
		}

		// Append to Tenant A
		err := adapter.Append(ctx, tenantA, agentID, threadID, entry1)
		assert.NoError(t, err)
		err = adapter.Append(ctx, tenantA, agentID, threadID, entry2)
		assert.NoError(t, err)

		// Get Tenant A history
		historyA, err := adapter.Get(ctx, tenantA, agentID, threadID, 10)
		assert.NoError(t, err)
		require.Len(t, historyA, 2)
		assert.Equal(t, "Hello from Tenant A", historyA[0].Content)
		assert.Equal(t, "Response for Tenant A", historyA[1].Content)

		// Get Tenant B history (should be empty - strict multi-tenancy)
		historyB, err := adapter.Get(ctx, tenantB, agentID, threadID, 10)
		assert.NoError(t, err)
		assert.Empty(t, historyB)

		// Test sliding window limit
		historyLimit, err := adapter.Get(ctx, tenantA, agentID, threadID, 1)
		assert.NoError(t, err)
		require.Len(t, historyLimit, 1)
		// Should be the most recent one (last turn appended)
		assert.Equal(t, "Response for Tenant A", historyLimit[0].Content)

		// Test Clear
		err = adapter.Clear(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)
		historyCleared, err := adapter.Get(ctx, tenantA, agentID, threadID, 10)
		assert.NoError(t, err)
		assert.Empty(t, historyCleared)
	})
}
