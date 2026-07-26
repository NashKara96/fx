package fx

import (
	"context"
	"errors"
)

// Hook defines a pair of start and stop callbacks for a lifecycle component.
// OnStart is called during application startup in registration order.
// OnStop is called during application shutdown in reverse registration order.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

// Lifecycle manages application lifecycle hooks. Hooks are registered
// via Append and executed by Start and Stop.
type Lifecycle struct {
	hooks []Hook
}

// NewLifecycle creates a new Lifecycle instance.
func NewLifecycle() *Lifecycle {
	return &Lifecycle{}
}

// Append registers one or more lifecycle hooks. They are executed in
// registration order on Start and in reverse order on Stop.
func (l *Lifecycle) Append(hooks ...Hook) {
	l.hooks = append(l.hooks, hooks...)
}

// Hooks returns a copy of the currently registered hooks.
func (l *Lifecycle) Hooks() []Hook {
	result := make([]Hook, len(l.hooks))
	copy(result, l.hooks)
	return result
}

// Start executes all registered OnStart hooks in registration order.
// If any hook returns an error, Start stops and returns that error;
// previously started hooks are NOT automatically stopped — the caller
// is responsible for calling Stop to clean up.
func (l *Lifecycle) Start(ctx context.Context) error {
	for _, hook := range l.hooks {
		if hook.OnStart != nil {
			if err := hook.OnStart(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// Stop executes all registered OnStop hooks in reverse registration
// order. It guarantees that every OnStop hook is attempted, even if
// one or more previous hooks returned an error (including
// context.Canceled or context.DeadlineExceeded).
//
// Errors from all failing hooks are accumulated and returned as a
// single combined error via errors.Join. If all hooks succeed, Stop
// returns nil.
func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error

	for i := len(l.hooks) - 1; i >= 0; i-- {
		if l.hooks[i].OnStop != nil {
			if err := l.hooks[i].OnStop(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}
