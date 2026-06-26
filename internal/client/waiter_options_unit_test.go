package client

import (
	"context"
	"errors"
	"testing"

	"github.com/sethvargo/go-retry"
)

// TestWaiterOptionsErrorIsNotRetryable pins that WaiterOptions.error wraps an
// error as NON-retryable: a retry.Do whose body returns options.error(...) must
// stop after exactly one attempt (it is the terminal-failure signal for every
// waiter loop). Kills a mutant that routes error() through RetryableError, which
// would make every permanent failure (4xx, decode, terminal state) loop until the
// backoff/context bound — masking real failures.
func TestWaiterOptionsErrorIsNotRetryable(t *testing.T) {
	opts := &WaiterOptions{}
	sentinel := errors.New("boom")
	calls := 0
	err := retry.Do(context.Background(), immediateBackoff(5), func(ctx context.Context) error {
		calls++
		return opts.error(sentinel)
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 (error() must be terminal, not retried)", calls)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error() must return the original error unchanged, got %v", err)
	}
}

// TestWaiterOptionsRetryableErrorIsRetryableAndWraps pins that
// WaiterOptions.retryableError makes an error retryable AND keeps the original
// reachable via errors.Is. The behavioral half (a retry.Do retries it then
// succeeds) kills a mutant that returns the bare error (no retry); the wrapping
// half kills a mutant that drops the cause.
func TestWaiterOptionsRetryableErrorIsRetryableAndWraps(t *testing.T) {
	opts := &WaiterOptions{}

	base := errors.New("transient cause")
	if !errors.Is(opts.retryableError(base), base) {
		t.Fatal("retryableError must keep the original error reachable via errors.Is")
	}

	calls := 0
	err := retry.Do(context.Background(), immediateBackoff(5), func(ctx context.Context) error {
		calls++
		if calls == 1 {
			return opts.retryableError(errors.New("transient"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("a retryableError must be retried then succeed, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2 (retryableError must be retried)", calls)
	}
}

// TestWaiterOptionsLogIsNilSafe pins that log never panics on a nil receiver or a
// nil Logger, and forwards the message to a configured Logger. nil-safety is
// asserted ONLY for log: error(nil)/retryableError(nil) would call err.Error() on
// a nil error and panic, but callers never pass a nil error, so that path is not a
// supported invariant and is deliberately not exercised.
func TestWaiterOptionsLogIsNilSafe(t *testing.T) {
	var nilOpts *WaiterOptions
	nilOpts.log("must not panic on a nil receiver")
	(&WaiterOptions{}).log("must not panic on a nil Logger")

	got := ""
	(&WaiterOptions{Logger: func(m string) { got = m }}).log("hello")
	if got != "hello" {
		t.Fatalf("a configured Logger must receive the message, got %q", got)
	}
}
