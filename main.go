package main

import (
	"context"
	"errors"
	"fmt"
)

// Hook represents a lifecycle hook function.
type Hook func(context.Context) error

// Lifecycle manages OnStart and OnStop hooks.
type Lifecycle struct {
	onStopHooks []Hook
}

// NewLifecycle creates a new Lifecycle.
func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

// AppendOnStop registers an OnStop hook. Hooks execute in reverse order.
func (l *Lifecycle) AppendOnStop(hook Hook) {
	l.onStopHooks = append(l.onStopHooks, hook)
}

// Stop executes all OnStop hooks in reverse registration order.
//
// If a hook returns an error (including context.Canceled or
// context.DeadlineExceeded), the error is accumulated but execution
// continues — remaining hooks are NOT skipped. All errors are combined
// and returned as a single aggregated error.
//
// The context is respected: if ctx is cancelled, hooks still run but
// each receives the cancelled context.
func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error

	for i := len(l.onStopHooks) - 1; i >= 0; i-- {
		if err := l.onStopHooks[i](ctx); err != nil {
			// Accumulate error but do NOT abort — continue executing remaining hooks
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// IsContextError returns true if the error is context.Canceled or context.DeadlineExceeded.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func main() {
	fmt.Println("Lifecycle with error accumulation.")
}
