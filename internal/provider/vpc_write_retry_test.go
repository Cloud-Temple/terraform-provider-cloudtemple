package provider

import (
	"context"
	"errors"
	"testing"
)

var (
	errVPCTransient = errors.New("transient platform failure")
	errVPCPermanent = errors.New("permanent failure")
)

// vpcIsTransient is the injected classifier for the mechanics tests: only the
// sentinel transient error is retryable. The real wiring uses
// client.IsTransientActivityFailure (covered by TestIsTransientActivityFailure).
func vpcIsTransient(err error) bool { return errors.Is(err, errVPCTransient) }

// scriptedAttempts replays the given per-attempt outcomes in order and counts
// the calls; the last outcome repeats if the loop attempts beyond the script.
func scriptedAttempts(calls *int, outcomes ...error) func(context.Context) error {
	return func(ctx context.Context) error {
		i := *calls
		if i >= len(outcomes) {
			i = len(outcomes) - 1
		}
		*calls++
		return outcomes[i]
	}
}

func countingSleep(sleeps *int) func(context.Context, int) error {
	return func(ctx context.Context, attempt int) error {
		*sleeps++
		return nil
	}
}

func TestRunVPCWriteWithRetry(t *testing.T) {
	t.Run("success on the first attempt: no retry, no sleep", func(t *testing.T) {
		calls, sleeps := 0, 0
		err := runVPCWriteWithRetry(context.Background(), vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, nil),
			sleep:       countingSleep(&sleeps),
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if err != nil {
			t.Fatalf("want success, got %v", err)
		}
		if calls != 1 || sleeps != 0 {
			t.Fatalf("calls=%d sleeps=%d, want 1/0", calls, sleeps)
		}
	})

	t.Run("transient then success: retried once", func(t *testing.T) {
		calls, sleeps := 0, 0
		err := runVPCWriteWithRetry(context.Background(), vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, errVPCTransient, nil),
			sleep:       countingSleep(&sleeps),
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if err != nil {
			t.Fatalf("want success after one transient retry, got %v", err)
		}
		if calls != 2 || sleeps != 1 {
			t.Fatalf("calls=%d sleeps=%d, want 2/1", calls, sleeps)
		}
	})

	t.Run("non-transient failure: returned immediately, never retried", func(t *testing.T) {
		calls, sleeps := 0, 0
		// The second outcome (nil) would make a WRONG retry succeed — proving
		// the helper does not retry a non-transient error.
		err := runVPCWriteWithRetry(context.Background(), vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, errVPCPermanent, nil),
			sleep:       countingSleep(&sleeps),
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if !errors.Is(err, errVPCPermanent) {
			t.Fatalf("want the permanent error, got %v", err)
		}
		if calls != 1 || sleeps != 0 {
			t.Fatalf("calls=%d sleeps=%d, want 1/0 (no retry on non-transient); kills the missing-isTransient-gate mutant", calls, sleeps)
		}
	})

	t.Run("always transient: bounded at maxTransientVPCAttempts", func(t *testing.T) {
		calls, sleeps := 0, 0
		err := runVPCWriteWithRetry(context.Background(), vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, errVPCTransient),
			sleep:       countingSleep(&sleeps),
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if !errors.Is(err, errVPCTransient) {
			t.Fatalf("want the last transient error returned, got %v", err)
		}
		if calls != maxTransientVPCAttempts || sleeps != maxTransientVPCAttempts-1 {
			t.Fatalf("calls=%d sleeps=%d, want %d/%d (bounded budget); kills the over-budget mutant",
				calls, sleeps, maxTransientVPCAttempts, maxTransientVPCAttempts-1)
		}
	})

	t.Run("context cancelled before the first attempt: never attempted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls, sleeps := 0, 0
		err := runVPCWriteWithRetry(ctx, vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, nil),
			sleep:       countingSleep(&sleeps),
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if err == nil {
			t.Fatal("want a context error")
		}
		if calls != 0 {
			t.Fatalf("calls=%d, want 0 (no attempt after cancellation)", calls)
		}
	})

	t.Run("cancellation during sleep stops without another attempt", func(t *testing.T) {
		calls := 0
		ctx, cancel := context.WithCancel(context.Background())
		cancelSleep := func(c context.Context, attempt int) error {
			cancel()
			return c.Err()
		}
		err := runVPCWriteWithRetry(ctx, vpcWriteRetry{
			attempt:     scriptedAttempts(&calls, errVPCTransient),
			sleep:       cancelSleep,
			isTransient: vpcIsTransient,
			label:       "test",
		})
		if err == nil {
			t.Fatal("want a cancellation error")
		}
		if calls != 1 {
			t.Fatalf("calls=%d, want 1 (cancelled during the first backoff, no second attempt); kills the cancel-during-sleep mutant", calls)
		}
	})
}
