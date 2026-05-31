package resiliency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/tacito-square/internal/agent/adapters/outbound/resiliency"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Execute(t *testing.T) {
	t.Run("should execute normal operation successfully when closed", func(t *testing.T) {
		cb := resiliency.NewCircuitBreaker(3, 100*time.Millisecond)

		called := false
		err := cb.Execute(context.Background(), func() error {
			called = true
			return nil
		}, func(err error) error {
			t.Fatal("fallback should not be called")
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("should execute fallback operation when operation fails and trip circuit", func(t *testing.T) {
		cb := resiliency.NewCircuitBreaker(2, 50*time.Millisecond)

		opErr := errors.New("operation failure")
		fallbackCalled := 0

		fallbackFn := func(err error) error {
			fallbackCalled++
			assert.ErrorIs(t, err, opErr)
			return errors.New("fallback response")
		}

		// First failure
		err := cb.Execute(context.Background(), func() error {
			return opErr
		}, fallbackFn)
		assert.EqualError(t, err, "fallback response")
		assert.Equal(t, 1, fallbackCalled)

		// Second failure (trips circuit)
		err = cb.Execute(context.Background(), func() error {
			return opErr
		}, fallbackFn)
		assert.EqualError(t, err, "fallback response")
		assert.Equal(t, 2, fallbackCalled)

		// Third call should bypass operation and trigger fallback with ErrCircuitOpen immediately
		opCalled := false
		err = cb.Execute(context.Background(), func() error {
			opCalled = true
			return nil
		}, func(err error) error {
			fallbackCalled++
			assert.ErrorIs(t, err, resiliency.ErrCircuitOpen)
			return errors.New("circuit bypassed")
		})

		assert.False(t, opCalled)
		assert.EqualError(t, err, "circuit bypassed")
		assert.Equal(t, 3, fallbackCalled)
	})

	t.Run("should recover state to closed when half-open trial succeeds", func(t *testing.T) {
		cb := resiliency.NewCircuitBreaker(1, 10*time.Millisecond)
		opErr := errors.New("db down")

		// Trip circuit
		_ = cb.Execute(context.Background(), func() error { return opErr }, func(err error) error { return err })

		// Verify circuit is open (fast-fails with ErrCircuitOpen)
		err := cb.Execute(context.Background(), func() error { return nil }, func(err error) error { return err })
		assert.ErrorIs(t, err, resiliency.ErrCircuitOpen)

		// Wait for sleep window to expire
		time.Sleep(15 * time.Millisecond)

		// Next request should transition to Half-Open and execute trial operation
		trialExecuted := false
		err = cb.Execute(context.Background(), func() error {
			trialExecuted = true
			return nil
		}, func(err error) error {
			return err
		})

		assert.NoError(t, err)
		assert.True(t, trialExecuted)

		// Circuit should be closed again, subsequent requests succeed without half-open transition
		err = cb.Execute(context.Background(), func() error { return nil }, func(err error) error { return err })
		assert.NoError(t, err)
	})
}
