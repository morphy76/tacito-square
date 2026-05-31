package resiliency

import (
	"context"
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	threshold       int
	recoveryTimeout time.Duration
	lastStateChange time.Time
}

func NewCircuitBreaker(threshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if recoveryTimeout <= 0 {
		recoveryTimeout = 15 * time.Second
	}
	return &CircuitBreaker{
		state:           StateClosed,
		threshold:       threshold,
		recoveryTimeout: recoveryTimeout,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, op func() error, fallback func(err error) error) error {
	if err := cb.beforeExecute(); err != nil {
		if fallback != nil {
			return fallback(err)
		}
		return err
	}

	err := op()
	cb.afterExecute(err)

	if err != nil {
		if fallback != nil {
			return fallback(err)
		}
		return err
	}

	return nil
}

func (cb *CircuitBreaker) beforeExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	if cb.state == StateOpen {
		if now.Sub(cb.lastStateChange) > cb.recoveryTimeout {
			cb.state = StateHalfOpen
			cb.lastStateChange = now
			return nil
		}
		return ErrCircuitOpen
	}
	return nil
}

func (cb *CircuitBreaker) afterExecute(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	if err != nil {
		cb.failures++
		if cb.state == StateHalfOpen || cb.failures >= cb.threshold {
			cb.state = StateOpen
			cb.lastStateChange = now
		}
	} else {
		cb.failures = 0
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
		}
	}
}
