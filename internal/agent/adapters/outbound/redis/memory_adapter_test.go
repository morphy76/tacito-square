//go:build integration

package redis

import (
	"context"
	"fmt"
	"sync"
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

		// Test RollbackLast
		// First reset and append two entries
		err = adapter.Clear(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)

		err = adapter.Append(ctx, tenantA, agentID, threadID, entry1)
		assert.NoError(t, err)
		err = adapter.Append(ctx, tenantA, agentID, threadID, entry2)
		assert.NoError(t, err)

		// Rollback once (removes entry2)
		err = adapter.RollbackLast(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)

		historyRollback1, err := adapter.Get(ctx, tenantA, agentID, threadID, 10)
		assert.NoError(t, err)
		require.Len(t, historyRollback1, 1)
		assert.Equal(t, "Hello from Tenant A", historyRollback1[0].Content)

		// Rollback again (removes entry1)
		err = adapter.RollbackLast(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)

		historyRollback2, err := adapter.Get(ctx, tenantA, agentID, threadID, 10)
		assert.NoError(t, err)
		assert.Empty(t, historyRollback2)

		// Rollback on empty key
		err = adapter.RollbackLast(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)

		// Test Clear
		err = adapter.Clear(ctx, tenantA, agentID, threadID)
		assert.NoError(t, err)
		historyCleared, err := adapter.Get(ctx, tenantA, agentID, threadID, 10)
		assert.NoError(t, err)
		assert.Empty(t, historyCleared)
	})

	t.Run("Lock and Unlock - ownership token verification", func(t *testing.T) {
		tenantA := "tenant-a"
		threadID := "thread-lock-1"

		// Acquire lock
		tokenA, locked, err := adapter.Lock(ctx, tenantA, threadID)
		assert.NoError(t, err)
		assert.True(t, locked)
		assert.NotEmpty(t, tokenA)

		// Try to acquire again (should fail/timeout since locked)
		// We set configuration to quick timeout for testing
		adapter.SetLockConfig(5*time.Second, 50*time.Millisecond)
		_, locked2, err2 := adapter.Lock(ctx, tenantA, threadID)
		assert.Error(t, err2)
		assert.False(t, locked2)

		// Try to unlock with a different/empty token (should fail)
		err = adapter.Unlock(ctx, tenantA, threadID, "wrong-token")
		assert.Error(t, err)

		// Unlock with correct token (should succeed)
		err = adapter.Unlock(ctx, tenantA, threadID, tokenA)
		assert.NoError(t, err)

		// Lock again after unlock (should succeed)
		tokenC, lockedC, errC := adapter.Lock(ctx, tenantA, threadID)
		assert.NoError(t, errC)
		assert.True(t, lockedC)
		_ = adapter.Unlock(ctx, tenantA, threadID, tokenC)
	})

	t.Run("Lock and Unlock - high-load concurrency mutual exclusion", func(t *testing.T) {
		tenantA := "tenant-a"
		threadID := "thread-lock-concurrency"
		adapter.SetLockConfig(5*time.Second, 5*time.Second)

		workers := 20
		acquiredCount := 0
		var mu sync.Mutex

		errChan := make(chan error, workers)
		var wg sync.WaitGroup

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				// Small random jitter before trying to lock
				time.Sleep(time.Duration(workerID) * 5 * time.Millisecond)

				// Attempt lock
				token, locked, err := adapter.Lock(ctx, tenantA, threadID)
				if err != nil {
					errChan <- err
					return
				}
				if !locked {
					errChan <- fmt.Errorf("worker %d failed to acquire lock", workerID)
					return
				}

				// Mutual Exclusion verification: increment count
				mu.Lock()
				acquiredCount++
				current := acquiredCount
				mu.Unlock()

				// If mutual exclusion is violated, count would be > 1 concurrently
				if current != 1 {
					_ = adapter.Unlock(ctx, tenantA, threadID, token)
					errChan <- fmt.Errorf("concurrency violation: worker %d acquired lock while others held it", workerID)
					return
				}

				// Simulate brief task duration
				time.Sleep(10 * time.Millisecond)

				// Decrement count
				mu.Lock()
				acquiredCount--
				mu.Unlock()

				// Unlock
				err = adapter.Unlock(ctx, tenantA, threadID, token)
				if err != nil {
					errChan <- err
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			assert.NoError(t, err)
		}
	})
}
