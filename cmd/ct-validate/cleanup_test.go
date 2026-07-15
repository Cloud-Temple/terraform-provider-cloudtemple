package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestCleanupLIFO proves teardowns run in reverse registration order. A
// mutation iterating forward instead of backward turns this RED.
func TestCleanupLIFO(t *testing.T) {
	c := NewCleanup()
	var order []string
	for _, name := range []string{"a", "b", "c"} {
		name := name
		c.Register(name, func(context.Context) error {
			order = append(order, name)
			return nil
		})
	}
	failures := c.TeardownAll(context.Background())
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %+v", failures)
	}
	want := []string{"c", "b", "a"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("teardown order = %v, want LIFO %v", order, want)
	}
}

// TestCleanupContinuesPastFailure proves a failing teardown does NOT stop the
// others: all three must run, and only the failing one is reported.
func TestCleanupContinuesPastFailure(t *testing.T) {
	c := NewCleanup()
	var ran []string
	c.Register("first", func(context.Context) error { ran = append(ran, "first"); return nil })
	// A 404 (permanent) is not retried, but must not abort the others.
	c.Register("boom", func(context.Context) error {
		ran = append(ran, "boom")
		return client.StatusError{Code: 404}
	})
	c.Register("last", func(context.Context) error { ran = append(ran, "last"); return nil })

	failures := c.TeardownAll(context.Background())

	// LIFO: last, boom, first — all must have run.
	want := []string{"last", "boom", "first"}
	if !reflect.DeepEqual(ran, want) {
		t.Fatalf("a failing teardown stopped the others: ran=%v want=%v", ran, want)
	}
	if len(failures) != 1 || failures[0].Label != "boom" {
		t.Fatalf("expected exactly the 'boom' failure, got %+v", failures)
	}
}

// TestCleanupBoundedTransientRetry proves a TRANSIENT teardown failure is
// retried up to teardownMaxAttempts and then succeeds, and that the retry is
// BOUNDED (no infinite loop). A permanent failure must not be retried.
func TestCleanupBoundedTransientRetry(t *testing.T) {
	// Transient then success: succeeds within the budget.
	t.Run("transient then success", func(t *testing.T) {
		attempts := 0
		c := NewCleanup()
		c.Register("flaky", func(context.Context) error {
			attempts++
			if attempts < teardownMaxAttempts {
				return client.StatusError{Code: 503} // transient 5xx
			}
			return nil
		})
		failures := c.TeardownAll(context.Background())
		if len(failures) != 0 {
			t.Fatalf("transient teardown should eventually succeed; failures=%+v", failures)
		}
		if attempts != teardownMaxAttempts {
			t.Fatalf("attempts=%d, want %d", attempts, teardownMaxAttempts)
		}
	})

	// Always transient: bounded — exactly teardownMaxAttempts, then reported.
	t.Run("always transient is bounded", func(t *testing.T) {
		attempts := 0
		c := NewCleanup()
		c.Register("always", func(context.Context) error {
			attempts++
			return client.StatusError{Code: 502}
		})
		failures := c.TeardownAll(context.Background())
		if attempts != teardownMaxAttempts {
			t.Fatalf("transient retry not bounded: attempts=%d, want %d", attempts, teardownMaxAttempts)
		}
		if len(failures) != 1 || failures[0].Attempts != teardownMaxAttempts {
			t.Fatalf("expected 1 failure after %d attempts, got %+v", teardownMaxAttempts, failures)
		}
	})

	// Permanent: NOT retried (exactly one attempt). A mutation making 4xx
	// retryable turns this RED.
	t.Run("permanent is not retried", func(t *testing.T) {
		attempts := 0
		c := NewCleanup()
		c.Register("perm", func(context.Context) error {
			attempts++
			return client.StatusError{Code: 400}
		})
		failures := c.TeardownAll(context.Background())
		if attempts != 1 {
			t.Fatalf("permanent failure must not be retried: attempts=%d", attempts)
		}
		if len(failures) != 1 {
			t.Fatalf("expected the permanent failure to be reported, got %+v", failures)
		}
	})
}

// TestCleanupCancelledContextStops proves a cancelled context stops teardown
// before running (no work attempted) rather than looping.
func TestCleanupCancelledContextStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ran := false
	c := NewCleanup()
	c.Register("x", func(context.Context) error { ran = true; return nil })
	failures := c.TeardownAll(ctx)
	if ran {
		t.Fatal("teardown ran despite a pre-cancelled context")
	}
	if len(failures) != 1 || !errors.Is(failures[0].Err, context.Canceled) {
		t.Fatalf("expected a context-cancelled failure, got %+v", failures)
	}
}

// TestCleanupDrains proves TeardownAll drains the tracker so a second call is a
// no-op (idempotent under the engine's defer + explicit call paths).
func TestCleanupDrains(t *testing.T) {
	c := NewCleanup()
	runs := 0
	c.Register("once", func(context.Context) error { runs++; return nil })
	if c.Pending() != 1 {
		t.Fatalf("Pending = %d, want 1", c.Pending())
	}
	c.TeardownAll(context.Background())
	c.TeardownAll(context.Background())
	if runs != 1 {
		t.Fatalf("teardown ran %d times, want exactly 1 (tracker must drain)", runs)
	}
	if c.Pending() != 0 {
		t.Fatalf("Pending = %d after teardown, want 0", c.Pending())
	}
}
