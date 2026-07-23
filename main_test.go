package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// Test 1: All hooks execute even when one returns context.Canceled
func TestStop_AllHooksExecuteOnContextCanceled(t *testing.T) {
	l := NewLifecycle()

	executed := []int{}
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 1)
		return nil
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 2)
		return context.Canceled
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 3)
		return nil
	})

	err := l.Stop(context.Background())

	// All hooks should have executed (in reverse order: 3, 2, 1)
	expectedOrder := []int{3, 2, 1}
	if len(executed) != 3 {
		t.Fatalf("expected 3 hooks to execute, got %d", len(executed))
	}
	for i, v := range executed {
		if v != expectedOrder[i] {
			t.Errorf("position %d: expected %d, got %d", i, expectedOrder[i], v)
		}
	}

	// Should return an error (the context.Canceled from hook 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error to contain context.Canceled, got %v", err)
	}
}

// Test 2: All hooks execute even when one returns context.DeadlineExceeded
func TestStop_AllHooksExecuteOnDeadlineExceeded(t *testing.T) {
	l := NewLifecycle()

	executed := []int{}
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 1)
		return nil
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 2)
		return context.DeadlineExceeded
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 3)
		return nil
	})

	err := l.Stop(context.Background())

	if len(executed) != 3 {
		t.Fatalf("expected 3 hooks to execute, got %d", len(executed))
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected error to contain context.DeadlineExceeded, got %v", err)
	}
}

// Test 3: Multiple errors are accumulated
func TestStop_MultipleErrorsAccumulated(t *testing.T) {
	l := NewLifecycle()

	customErr1 := errors.New("custom error 1")
	customErr2 := errors.New("custom error 2")

	l.AppendOnStop(func(ctx context.Context) error {
		return customErr1
	})
	l.AppendOnStop(func(ctx context.Context) error {
		return context.Canceled
	})
	l.AppendOnStop(func(ctx context.Context) error {
		return customErr2
	})

	err := l.Stop(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, customErr1) {
		t.Errorf("error should contain customErr1, got %v", err)
	}
	if !errors.Is(err, customErr2) {
		t.Errorf("error should contain customErr2, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error should contain context.Canceled, got %v", err)
	}
}

// Test 4: Reverse order is strictly preserved
func TestStop_ReverseOrderPreserved(t *testing.T) {
	l := NewLifecycle()

	executed := []int{}
	for i := 1; i <= 5; i++ {
		idx := i
		l.AppendOnStop(func(ctx context.Context) error {
			executed = append(executed, idx)
			return nil
		})
	}

	l.Stop(context.Background())

	expected := []int{5, 4, 3, 2, 1}
	for i, v := range executed {
		if v != expected[i] {
			t.Errorf("position %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

// Test 5: No hooks returns nil
func TestStop_NoHooksReturnsNil(t *testing.T) {
	l := NewLifecycle()
	err := l.Stop(context.Background())
	if err != nil {
		t.Errorf("expected nil error for no hooks, got %v", err)
	}
}

// Test 6: All hooks execute when all return errors
func TestStop_AllHooksExecuteWhenAllFail(t *testing.T) {
	l := NewLifecycle()

	executed := []int{}
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 1)
		return errors.New("err1")
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 2)
		return errors.New("err2")
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 3)
		return errors.New("err3")
	})

	err := l.Stop(context.Background())

	if len(executed) != 3 {
		t.Fatalf("expected 3 hooks to execute, got %d", len(executed))
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Test 7: First hook (last in execution) returning context.Canceled doesn't block others
func TestStop_FirstHookErrorDoesNotBlockOthers(t *testing.T) {
	l := NewLifecycle()

	executed := []int{}
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 1)
		return context.Canceled
	})
	l.AppendOnStop(func(ctx context.Context) error {
		executed = append(executed, 2)
		return nil
	})

	err := l.Stop(context.Background())

	if len(executed) != 2 {
		t.Fatalf("expected 2 hooks to execute, got %d", len(executed))
	}
	// Reverse order: 2 first, then 1
	if executed[0] != 2 || executed[1] != 1 {
		t.Errorf("expected order [2, 1], got %v", executed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in error, got %v", err)
	}
}

// Test 8: IsContextError helper
func TestIsContextError(t *testing.T) {
	if !IsContextError(context.Canceled) {
		t.Error("context.Canceled should be detected")
	}
	if !IsContextError(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should be detected")
	}
	if IsContextError(errors.New("custom error")) {
		t.Error("custom error should not be detected as context error")
	}
	// Wrapped context errors
	wrapped := fmt.Errorf("wrapped: %w", context.Canceled) //nolint:govet // intentional wrap for test
	if !IsContextError(wrapped) {
		t.Error("wrapped context.Canceled should be detected")
	}
}
