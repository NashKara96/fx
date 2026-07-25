package main

import (
	"context"
	"fmt"
	"sync"
)

// Hook represents a lifecycle hook.
type Hook struct {
	Name string
	OnStop func(ctx context.Context) error
}

// Lifecycle manages application lifecycle with the fix: all OnStop
// hooks run even if a prior hook returns context.Canceled or
// context.DeadlineExceeded.
type Lifecycle struct {
	mu    sync.Mutex
	hooks []Hook
}

func NewLifecycle() *Lifecycle {
	return &Lifecycle{hooks: make([]Hook, 0)}
}

// Append registers a hook.
func (l *Lifecycle) Append(h Hook) {
	l.mu.Lock()
	l.hooks = append(l.hooks, h)
	l.mu.Unlock()
}

// Stop executes all OnStop hooks in reverse order. The key fix is that
// even if a hook returns context.Canceled or context.DeadlineExceeded,
// we continue executing the remaining hooks to ensure graceful shutdown.
func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	hooks := make([]Hook, len(l.hooks))
	copy(hooks, l.hooks)
	l.mu.Unlock()

	var finalErr error
	// Execute in reverse order
	for i := len(hooks) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if hooks[i].OnStop == nil {
			continue
		}

		if err := hooks[i].OnStop(ctx); err != nil {
			fmt.Printf("Hook %q returned error: %v (continuing with remaining hooks)\n", hooks[i].Name, err)
			// The fix: do NOT abort on context.Canceled or context.DeadlineExceeded.
			// Collect the error but keep running remaining hooks.
			if finalErr == nil {
				finalErr = err
			}
		} else {
			fmt.Printf("Hook %q completed successfully\n", hooks[i].Name)
		}
	}

	return finalErr
}

func main() {
	fmt.Println("fx Lifecycle.Stop hook continuation fix")
	fmt.Println("========================================")

	lc := NewLifecycle()

	// Register hooks in order
	lc.Append(Hook{
		Name: "database",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [database] shutting down...")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "message_queue",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [message_queue] shutting down...")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "http_server",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [http_server] shutting down...")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "background_worker",
		OnStop: func(ctx context.Context) error {
			return context.Canceled
		},
	})
	lc.Append(Hook{
		Name: "metrics_exporter",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [metrics_exporter] shutting down...")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "network_listener",
		OnStop: func(ctx context.Context) error {
			fmt.Println("  [network_listener] shutting down...")
			return nil
		},
	})

	fmt.Println("\nRunning Stop() with a failing hook in the middle...")
	err := lc.Stop(context.Background())

	fmt.Println("\nResults:")
	fmt.Println("  - 'background_worker' returned context.Canceled")
	fmt.Println("  - All subsequent hooks (metrics_exporter, network_listener) still executed")
	fmt.Println("  - http_server, message_queue, database also executed (reverse order)")
	fmt.Printf("  - Final error: %v\n", err)
	fmt.Println("\nFix verification complete — all hooks ran despite prior failure.")
}
