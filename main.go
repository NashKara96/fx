package main

import (
	"context"
	"fmt"
	"sync"
)

// Lifecycle ensures all OnStop hooks execute even if prior hooks return errors.
// The bug: if a hook returns context.Canceled, subsequent hooks are skipped.
// Fix: Collect errors but continue executing all hooks.
type Lifecycle struct {
	mu     sync.Mutex
	stops  []func(ctx context.Context) error
}

func (l *Lifecycle) Append(stop func(ctx context.Context) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stops = append(l.stops, stop)
}

func (l *Lifecycle) Stop(ctx context.Context) []error {
	l.mu.Lock()
	stops := l.stops
	l.stops = nil
	l.mu.Unlock()

	var errs []error
	for i := len(stops) - 1; i >= 0; i-- {
		if err := stops[i](ctx); err != nil {
			errs = append(errs, err)
			// Continue to next hook even if this one failed
		}
	}
	return errs
}

func main() {
	l := &Lifecycle{}
	l.Append(func(ctx context.Context) error { return context.Canceled })
	l.Append(func(ctx context.Context) error { return nil })
	l.Append(func(ctx context.Context) error { return nil })
	errs := l.Stop(context.Background())
	fmt.Printf("Errors: %d, all hooks executed\n", len(errs))
}
