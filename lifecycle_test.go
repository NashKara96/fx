package fx

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

// TestStop_AllHooksExecuted verifies that all registered OnStop hooks
// are called during Stop(), even when some of them return errors.
func TestStop_AllHooksExecuted(t *testing.T) {
	lc := NewLifecycle()

	var callOrder []int
	stop1 := func(ctx context.Context) error {
		callOrder = append(callOrder, 1)
		return nil
	}
	stop2 := func(ctx context.Context) error {
		callOrder = append(callOrder, 2)
		return errors.New("hook 2 failed")
	}
	stop3 := func(ctx context.Context) error {
		callOrder = append(callOrder, 3)
		return nil
	}

	lc.Append(
		Hook{OnStop: stop1},
		Hook{OnStop: stop2},
		Hook{OnStop: stop3},
	)

	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	// Verify all 3 hooks were executed (reverse order: 3, 2, 1)
	if len(callOrder) != 3 {
		t.Fatalf("expected 3 hook calls, got %d: %v", len(callOrder), callOrder)
	}

	// Verify reverse execution order: hook 3 (last registered) executes first
	if callOrder[0] != 3 {
		t.Errorf("expected first hook called to be 3 (last registered), got %d", callOrder[0])
	}
	if callOrder[1] != 2 {
		t.Errorf("expected second hook called to be 2, got %d", callOrder[1])
	}
	if callOrder[2] != 1 {
		t.Errorf("expected third hook called to be 1 (first registered), got %d", callOrder[2])
	}

	// Verify the error from hook 2 is present in the combined error
	if !strings.Contains(err.Error(), "hook 2 failed") {
		t.Errorf("expected combined error to contain 'hook 2 failed', got: %v", err)
	}
}

// TestStop_ContextCanceledDoesNotBlockOtherHooks ensures that when a hook
// returns context.Canceled, the remaining hooks are still executed.
func TestStop_ContextCanceledDoesNotBlockOtherHooks(t *testing.T) {
	lc := NewLifecycle()

	var executedCount atomic.Int32

	hook1 := Hook{
		OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return context.Canceled
		},
	}
	hook2 := Hook{
		OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		},
	}
	hook3 := Hook{
		OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		},
	}

	lc.Append(hook1, hook2, hook3)

	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	if !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Errorf("expected combined error to contain 'context canceled', got: %v", err)
	}

	if count := executedCount.Load(); count != 3 {
		t.Errorf("expected all 3 hooks to execute, but only %d executed", count)
	}
}

// TestStop_ContextDeadlineExceededDoesNotBlockOtherHooks ensures that when
// a hook returns context.DeadlineExceeded, the remaining hooks are still executed.
func TestStop_ContextDeadlineExceededDoesNotBlockOtherHooks(t *testing.T) {
	lc := NewLifecycle()

	var executedCount atomic.Int32

	lc.Append(
		Hook{OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return context.DeadlineExceeded
		}},
		Hook{OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		}},
	)

	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Errorf("expected combined error to contain 'context deadline exceeded', got: %v", err)
	}

	if count := executedCount.Load(); count != 2 {
		t.Errorf("expected both hooks to execute, but only %d executed", count)
	}
}

// TestStop_ReverseOrder verifies that OnStop hooks are called in reverse
// registration order.
func TestStop_ReverseOrder(t *testing.T) {
	lc := NewLifecycle()

	var order []int
	lc.Append(
		Hook{OnStop: func(ctx context.Context) error {
			order = append(order, 1)
			return nil
		}},
		Hook{OnStop: func(ctx context.Context) error {
			order = append(order, 2)
			return nil
		}},
		Hook{OnStop: func(ctx context.Context) error {
			order = append(order, 3)
			return nil
		}},
	)

	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int{3, 2, 1}
	if !equalSlice(order, want) {
		t.Errorf("expected order %v, got %v", want, order)
	}
}

// TestStop_NoHooks verifies that Stop works correctly when no hooks are
// registered.
func TestStop_NoHooks(t *testing.T) {
	lc := NewLifecycle()

	if err := lc.Stop(context.Background()); err != nil {
		t.Errorf("expected nil error with no hooks, got: %v", err)
	}
}

// TestStop_MultipleErrors verifies that when multiple hooks fail, all
// their errors are accumulated and returned.
func TestStop_MultipleErrors(t *testing.T) {
	lc := NewLifecycle()

	lc.Append(
		Hook{OnStop: func(ctx context.Context) error {
			return errors.New("error from hook A")
		}},
		Hook{OnStop: func(ctx context.Context) error {
			return errors.New("error from hook B")
		}},
	)

	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	if !strings.Contains(err.Error(), "error from hook A") {
		t.Errorf("expected combined error to contain 'error from hook A', got: %v", err)
	}
	if !strings.Contains(err.Error(), "error from hook B") {
		t.Errorf("expected combined error to contain 'error from hook B', got: %v", err)
	}
}

// TestStop_AllSucceed verifies that Stop returns nil when all hooks succeed.
func TestStop_AllSucceed(t *testing.T) {
	lc := NewLifecycle()

	lc.Append(
		Hook{OnStop: func(ctx context.Context) error {
			return nil
		}},
		Hook{OnStop: func(ctx context.Context) error {
			return nil
		}},
	)

	if err := lc.Stop(context.Background()); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

// TestStop_NilOnStopFields verifies that hooks with nil OnStop fields
// are safely skipped.
func TestStop_NilOnStopFields(t *testing.T) {
	lc := NewLifecycle()

	var executedCount atomic.Int32

	lc.Append(
		Hook{OnStart: func(ctx context.Context) error {
			return nil
		}}, // nil OnStop, should be skipped
		Hook{OnStop: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		}},
	)

	if err := lc.Stop(context.Background()); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	if count := executedCount.Load(); count != 1 {
		t.Errorf("expected 1 hook to execute, got %d", count)
	}
}

// TestStart_StopsOnError verifies that Start stops at the first error
// and does not execute subsequent OnStart hooks.
func TestStart_StopsOnError(t *testing.T) {
	lc := NewLifecycle()

	var executedCount atomic.Int32

	lc.Append(
		Hook{OnStart: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		}},
		Hook{OnStart: func(ctx context.Context) error {
			executedCount.Add(1)
			return errors.New("start failed")
		}},
		Hook{OnStart: func(ctx context.Context) error {
			executedCount.Add(1)
			return nil
		}},
	)

	err := lc.Start(context.Background())
	if err == nil {
		t.Fatal("expected an error but got nil")
	}

	if !strings.Contains(err.Error(), "start failed") {
		t.Errorf("expected error 'start failed', got: %v", err)
	}

	// Only 2 hooks should have executed (first succeeds, second fails, third skipped)
	if count := executedCount.Load(); count != 2 {
		t.Errorf("expected 2 hooks to execute, got %d", count)
	}
}

// TestIntegration_StartStop verifies a full start-stop lifecycle with
// multiple components.
func TestIntegration_StartStop(t *testing.T) {
	lc := NewLifecycle()

	var startOrder []int
	var stopOrder []int

	lc.Append(
		Hook{
			OnStart: func(ctx context.Context) error {
				startOrder = append(startOrder, 1)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				stopOrder = append(stopOrder, 1)
				return nil
			},
		},
		Hook{
			OnStart: func(ctx context.Context) error {
				startOrder = append(startOrder, 2)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				stopOrder = append(stopOrder, 2)
				return errors.New("stop error 2")
			},
		},
		Hook{
			OnStart: func(ctx context.Context) error {
				startOrder = append(startOrder, 3)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				stopOrder = append(stopOrder, 3)
				return nil
			},
		},
	)

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	wantStart := []int{1, 2, 3}
	if !equalSlice(startOrder, wantStart) {
		t.Errorf("Start order: expected %v, got %v", wantStart, startOrder)
	}

	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected an error from Stop but got nil")
	}

	wantStop := []int{3, 2, 1}
	if !equalSlice(stopOrder, wantStop) {
		t.Errorf("Stop order: expected %v, got %v", wantStop, stopOrder)
	}

	if !strings.Contains(err.Error(), "stop error 2") {
		t.Errorf("expected combined error to contain 'stop error 2', got: %v", err)
	}
}

// equalSlice is a helper to compare two int slices.
func equalSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
