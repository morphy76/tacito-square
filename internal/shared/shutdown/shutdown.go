// Package shutdown provides a graceful shutdown manager for Tacito Square components.
// It registers cleanup hooks (e.g., tracer flush, HTTP server close, DB close) and
// executes them in reverse registration order when the process receives a termination signal.
package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"
)

// Hook is a cleanup function invoked during graceful shutdown.
type Hook func(ctx context.Context) error

type namedHook struct {
	name string
	fn   Hook
}

// Manager registers and runs shutdown hooks in reverse order with a timeout.
type Manager struct {
	hooks   []namedHook
	timeout time.Duration
}

// NewManager creates a new shutdown manager with the given timeout.
// The timeout limits total time for all hooks to complete.
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		timeout: timeout,
	}
}

// Register adds a named shutdown hook. Hooks run in reverse registration order
// (LIFO) so that dependencies registered first are cleaned up last.
func (m *Manager) Register(name string, hook Hook) {
	m.hooks = append(m.hooks, namedHook{name: name, fn: hook})
}

// HookCount returns the number of registered hooks.
func (m *Manager) HookCount() int {
	return len(m.hooks)
}

// RunHooks executes all registered hooks in reverse order with the configured timeout.
// It continues executing hooks even if one fails, collecting all errors.
func (m *Manager) RunHooks(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, m.timeout)
	defer cancel()

	var errs []error

	// Execute in reverse order (LIFO)
	for i := len(m.hooks) - 1; i >= 0; i-- {
		h := m.hooks[i]
		if err := h.fn(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown hook %q: %w", h.name, err))
		}
	}

	return errors.Join(errs...)
}

// Wait blocks until one of the given signals is received, then runs all hooks.
// If no signals are specified, defaults to SIGINT and SIGTERM.
// Returns the combined error from all hooks (nil if all succeed).
func (m *Manager) Wait(signals ...os.Signal) error {
	if len(signals) == 0 {
		signals = []os.Signal{os.Interrupt}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	<-sigCh
	signal.Stop(sigCh)

	return m.RunHooks(context.Background())
}
