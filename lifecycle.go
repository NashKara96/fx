package fx

import (
	"context"
	"errors"
	"sync"
)

// Hook represents a lifecycle hook with an OnStop callback.
type Hook struct {
	// Name is an optional identifier for the hook (used for debugging).
	Name string
	// OnStop is called during shutdown. Errors are collected but do not
	// prevent remaining hooks from executing.
	OnStop func(ctx context.Context) error
}

// Lifecycle manages application lifecycle hooks (OnStop).
// Hooks are executed in reverse registration order during Stop().
type Lifecycle struct {
	mu    sync.Mutex
	hooks []Hook
}

// NewLifecycle creates a new Lifecycle instance.
func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

// Append registers a new Hook. Hooks are executed in reverse order
// during Stop (LIFO).
func (l *Lifecycle) Append(h Hook) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = append(l.hooks, h)
}

// Stop executes all registered OnStop hooks in reverse order and
// collects any errors they return.
//
// The fix: even if a hook returns an error (including context.Canceled
// or context.DeadlineExceeded), execution continues so that every
// registered cleanup hook gets a chance to run.
//
// If the context passed to Stop is cancelled or exceeds its deadline,
// the method returns the context error immediately. Individual hook
// errors are accumulated and returned as a single combined error.
func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	// Copy the hooks slice under the lock so the loop is safe from
	// concurrent Append calls.
	hooks := make([]Hook, len(l.hooks))
	copy(hooks, l.hooks)
	l.mu.Unlock()

	var errs []error

	// Execute in reverse registration order (LIFO).
	for i := len(hooks) - 1; i >= 0; i-- {
		// Respect parent context cancellation — if the caller's context
		// has been cancelled or timed out, abort immediately.
		select {
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		default:
		}

		if hooks[i].OnStop == nil {
			continue
		}

		if err := hooks[i].OnStop(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
