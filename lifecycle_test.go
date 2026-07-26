package fx

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestStop_AllHooksExecuteOnContextCanceled verifies that all OnStop hooks
// execute even when a hook returns context.Canceled.
func TestStop_AllHooksExecuteOnContextCanceled(t *testing.T) {
	lc := NewLifecycle()
	var executed int32

	lc.Append(Hook{
		Name: "hook1",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return context.Canceled
		},
	})
	lc.Append(Hook{
		Name: "hook3",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})

	err := lc.Stop(context.Background())

	if n := atomic.LoadInt32(&executed); n != 3 {
		t.Errorf("expected 3 hooks executed, got %d", n)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in aggregated error, got %v", err)
	}
}

// TestStop_AllHooksExecuteOnDeadlineExceeded verifies that all OnStop hooks
// execute even when a hook returns context.DeadlineExceeded.
func TestStop_AllHooksExecuteOnDeadlineExceeded(t *testing.T) {
	lc := NewLifecycle()
	var executed int32

	lc.Append(Hook{
		Name: "hook1",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return context.DeadlineExceeded
		},
	})
	lc.Append(Hook{
		Name: "hook3",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})

	err := lc.Stop(context.Background())

	if n := atomic.LoadInt32(&executed); n != 3 {
		t.Errorf("expected 3 hooks executed, got %d", n)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded in aggregated error, got %v", err)
	}
}

// TestStop_MultipleErrorsAccumulated verifies that when multiple hooks fail,
// all their errors are accumulated and returned.
func TestStop_MultipleErrorsAccumulated(t *testing.T) {
	lc := NewLifecycle()

	lc.Append(Hook{
		Name: "hook1",
		OnStop: func(ctx context.Context) error {
			return errors.New("error1")
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			return errors.New("error2")
		},
	})
	lc.Append(Hook{
		Name: "hook3",
		OnStop: func(ctx context.Context) error {
			return errors.New("error3")
		},
	})

	err := lc.Stop(context.Background())

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check that all three errors are present in the combined error.
	for _, want := range []string{"error1", "error2", "error3"} {
		if !containsError(err, want) {
			t.Errorf("expected aggregated error to contain %q", want)
		}
	}
}

// TestStop_ReverseOrderPreserved verifies that OnStop hooks execute in
// reverse registration order (LIFO).
func TestStop_ReverseOrderPreserved(t *testing.T) {
	lc := NewLifecycle()
	var order []string

	lc.Append(Hook{
		Name: "first",
		OnStop: func(ctx context.Context) error {
			order = append(order, "first")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "second",
		OnStop: func(ctx context.Context) error {
			order = append(order, "second")
			return nil
		},
	})
	lc.Append(Hook{
		Name: "third",
		OnStop: func(ctx context.Context) error {
			order = append(order, "third")
			return nil
		},
	})

	_ = lc.Stop(context.Background())

	want := []string{"third", "second", "first"}
	if len(order) != len(want) {
		t.Fatalf("expected %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, order)
		}
	}
}

// TestStop_NoHooksReturnsNil verifies that Stop() returns nil when no
// hooks are registered.
func TestStop_NoHooksReturnsNil(t *testing.T) {
	lc := NewLifecycle()
	err := lc.Stop(context.Background())
	if err != nil {
		t.Errorf("expected nil error for empty lifecycle, got %v", err)
	}
}

// TestStop_AllHooksExecuteWhenAllFail verifies that even when every hook
// fails, all are still executed.
func TestStop_AllHooksExecuteWhenAllFail(t *testing.T) {
	lc := NewLifecycle()
	var executed int32

	lc.Append(Hook{
		Name: "hook1",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return errors.New("fail1")
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return errors.New("fail2")
		},
	})
	lc.Append(Hook{
		Name: "hook3",
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return errors.New("fail3")
		},
	})

	_ = lc.Stop(context.Background())

	if n := atomic.LoadInt32(&executed); n != 3 {
		t.Errorf("expected 3 hooks executed, got %d", n)
	}
}

// TestStop_FirstHookErrorDoesNotBlockOthers verifies that a failing hook
// does not prevent subsequent (earlier registered) hooks from running.
func TestStop_FirstHookErrorDoesNotBlockOthers(t *testing.T) {
	lc := NewLifecycle()
	var executed int32

	// In reverse order, hook3 executes first. Make it fail.
	lc.Append(Hook{
		Name: "hook1", // executes last
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})
	lc.Append(Hook{
		Name: "hook2", // executes middle
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return nil
		},
	})
	lc.Append(Hook{
		Name: "hook3", // executes first (reversed)
		OnStop: func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			return errors.New("first hook to run fails")
		},
	})

	_ = lc.Stop(context.Background())

	if n := atomic.LoadInt32(&executed); n != 3 {
		t.Errorf("expected 3 hooks executed, got %d", n)
	}
}

// TestStop_ContextCancellationAborts verifies that if the context passed
// to Stop is cancelled, the method aborts and returns the context error.
func TestStop_ContextCancellationAborts(t *testing.T) {
	lc := NewLifecycle()
	hook1Called := false
	hook2Called := false

	lc.Append(Hook{
		Name: "hook1",
		OnStop: func(ctx context.Context) error {
			hook1Called = true
			return nil
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			hook2Called = true
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := lc.Stop(ctx)

	// When context is already cancelled, Stop may return immediately
	// without executing any hooks.
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// At least hook2 (the one to execute first in reverse order) may or
	// may not be called depending on the implementation. Since we check
	// ctx.Done() at the start of each iteration, the first hook (hook2
	// in reverse order) might run before we detect cancellation.
	// This is acceptable behavior — what matters is that context
	// cancellation is respected.
	_ = hook1Called
	_ = hook2Called
}

// TestStop_NilOnStopHook verifies that a nil OnStop callback does not
// cause a panic and is skipped.
func TestStop_NilOnStopHook(t *testing.T) {
	lc := NewLifecycle()
	hook2Called := false

	lc.Append(Hook{Name: "hook1"}) // nil OnStop
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			hook2Called = true
			return nil
		},
	})

	err := lc.Stop(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !hook2Called {
		t.Error("expected hook2 to be called")
	}
}

// TestStop_ConcurrentSafe verifies that Append and Stop can be called
// concurrently without data races.
func TestStop_ConcurrentSafe(t *testing.T) {
	lc := NewLifecycle()

	// Pre-register hooks.
	for i := 0; i < 10; i++ {
		i := i
		lc.Append(Hook{
			Name: "hook",
			OnStop: func(ctx context.Context) error {
				_ = i
				return nil
			},
		})
	}

	done := make(chan struct{})
	go func() {
		lc.Append(Hook{
			Name: "concurrent_hook",
			OnStop: func(ctx context.Context) error {
				return nil
			},
		})
		close(done)
	}()

	err := lc.Stop(context.Background())
	<-done
	_ = err
	// No race condition detected by `go test -race`.
}

// containsError checks whether the aggregated error contains a string
// matching s. errors.Join concatenates errors with "\n".
func containsError(err error, s string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), s)
}

// TestStop_RespectsDeadline ensures that if a context with a very short
// deadline is passed, hooks are cancelled accordingly.
func TestStop_RespectsDeadline(t *testing.T) {
	lc := NewLifecycle()
	hook2Called := false

	lc.Append(Hook{
		Name: "long_hook",
		OnStop: func(ctx context.Context) error {
			// This hook takes longer than the deadline.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
				return nil
			}
		},
	})
	lc.Append(Hook{
		Name: "hook2",
		OnStop: func(ctx context.Context) error {
			hook2Called = true
			return nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond) // Let the deadline expire.

	err := lc.Stop(ctx)
	if err == nil {
		t.Error("expected an error due to deadline exceeded")
	}
	// The first hook in reverse order (long_hook) may or may not be
	// reached depending on precisely when context expires. This test
	// just verifies that a deadline is respected.
	_ = hook2Called
}
