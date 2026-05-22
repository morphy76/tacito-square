package shutdown

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_CreatesManager(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	require.NotNil(t, mgr)
}

func TestManager_Register_AddsHook(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	mgr.Register("test-hook", func(ctx context.Context) error {
		return nil
	})
	assert.Equal(t, 1, mgr.HookCount())
}

func TestManager_RunHooks_ExecutesAllHooks(t *testing.T) {
	mgr := NewManager(5 * time.Second)

	var counter int32
	mgr.Register("hook-1", func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})
	mgr.Register("hook-2", func(ctx context.Context) error {
		atomic.AddInt32(&counter, 1)
		return nil
	})

	err := mgr.RunHooks(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&counter))
}

func TestManager_RunHooks_ExecutesInReverseOrder(t *testing.T) {
	mgr := NewManager(5 * time.Second)

	var order []string
	mgr.Register("first", func(ctx context.Context) error {
		order = append(order, "first")
		return nil
	})
	mgr.Register("second", func(ctx context.Context) error {
		order = append(order, "second")
		return nil
	})
	mgr.Register("third", func(ctx context.Context) error {
		order = append(order, "third")
		return nil
	})

	err := mgr.RunHooks(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []string{"third", "second", "first"}, order)
}

func TestManager_RunHooks_RespectsTimeout(t *testing.T) {
	mgr := NewManager(100 * time.Millisecond)

	mgr.Register("slow-hook", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	err := mgr.RunHooks(context.Background())
	assert.Error(t, err) // should error due to timeout
}

func TestManager_RunHooks_ContinuesOnError(t *testing.T) {
	mgr := NewManager(5 * time.Second)

	var secondRan bool
	mgr.Register("failing-hook", func(ctx context.Context) error {
		return assert.AnError
	})
	mgr.Register("second-hook", func(ctx context.Context) error {
		secondRan = true
		return nil
	})

	err := mgr.RunHooks(context.Background())
	// Should still run the second hook (registered first, runs second in reverse)
	// and report the error
	assert.Error(t, err)
	assert.True(t, secondRan)
}

func TestManager_RunHooks_NoHooks_Succeeds(t *testing.T) {
	mgr := NewManager(5 * time.Second)
	err := mgr.RunHooks(context.Background())
	assert.NoError(t, err)
}
