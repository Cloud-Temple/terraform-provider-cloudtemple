package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/sethvargo/go-retry"
)

// immediateBackoff never sleeps: the polling loop is driven entirely by the
// injected reads, bounded by maxRetries to keep every test terminating.
func immediateBackoff(maxRetries uint64) retry.Backoff {
	return retry.WithMaxRetries(maxRetries, retry.BackoffFunc(func() (time.Duration, bool) {
		return 0, false
	}))
}

// scriptedReads returns an activityReadFunc that replays the given outcomes
// in order and counts the calls. The last outcome repeats if the loop polls
// beyond the script.
type readOutcome struct {
	activity *Activity
	err      error
}

func scriptedReads(calls *int, outcomes ...readOutcome) activityReadFunc {
	return func(ctx context.Context) (*Activity, error) {
		i := *calls
		if i >= len(outcomes) {
			i = len(outcomes) - 1
		}
		*calls++
		return outcomes[i].activity, outcomes[i].err
	}
}

func completedActivity() *Activity {
	return &Activity{ID: "act-1", State: map[string]ActivityState{"completed": {}}}
}

func failedActivity() *Activity {
	return &Activity{ID: "act-1", State: map[string]ActivityState{"failed": {}}}
}

func pendingActivity() *Activity {
	return &Activity{ID: "act-1", State: map[string]ActivityState{"pending": {}}}
}

func TestActivityWaitTransientReadErrorsAreRetried(t *testing.T) {
	transients := []error{
		StatusError{Code: http.StatusTooManyRequests},
		StatusError{Code: http.StatusInternalServerError},
		StatusError{Code: http.StatusBadGateway},
		StatusError{Code: http.StatusServiceUnavailable},
		StatusError{Code: http.StatusGatewayTimeout},
		&url.Error{Op: "Get", URL: "https://shiva.example", Err: errors.New("connection reset")},
	}
	for _, transient := range transients {
		t.Run(fmt.Sprintf("%v", transient), func(t *testing.T) {
			calls := 0
			read := scriptedReads(&calls,
				readOutcome{err: transient},
				readOutcome{err: transient},
				readOutcome{activity: completedActivity()},
			)
			activity, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
			if err != nil {
				t.Fatalf("transient errors must be survived, got: %s", err)
			}
			if activity == nil || calls != 3 {
				t.Fatalf("activity=%v calls=%d, want completed after 3 reads", activity, calls)
			}
		})
	}
}

func TestActivityWaitTransientBudgetIsBoundedAndConsecutive(t *testing.T) {
	t.Run("permanent failure after the consecutive budget is exhausted", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls, readOutcome{err: StatusError{Code: http.StatusInternalServerError}})
		_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(50), nil)
		if err == nil {
			t.Fatal("an uninterrupted stream of 500s must eventually fail")
		}
		// initial attempt + maxActivityReadRetries tolerated retries
		if calls != maxActivityReadRetries+1 {
			t.Fatalf("calls=%d, want %d (bounded budget)", calls, maxActivityReadRetries+1)
		}
	})

	t.Run("a successful read resets the consecutive budget", func(t *testing.T) {
		calls := 0
		outcomes := []readOutcome{}
		// maxActivityReadRetries failures, one successful pending read,
		// maxActivityReadRetries failures again: must still be alive, then
		// complete.
		for i := 0; i < maxActivityReadRetries; i++ {
			outcomes = append(outcomes, readOutcome{err: StatusError{Code: http.StatusInternalServerError}})
		}
		outcomes = append(outcomes, readOutcome{activity: pendingActivity()})
		for i := 0; i < maxActivityReadRetries; i++ {
			outcomes = append(outcomes, readOutcome{err: StatusError{Code: http.StatusServiceUnavailable}})
		}
		outcomes = append(outcomes, readOutcome{activity: completedActivity()})
		read := scriptedReads(&calls, outcomes...)
		activity, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(50), nil)
		if err != nil || activity == nil {
			t.Fatalf("interleaved successes must reset the budget, got err=%v", err)
		}
		if calls != 2*maxActivityReadRetries+2 {
			t.Fatalf("calls=%d, want %d", calls, 2*maxActivityReadRetries+2)
		}
	})
}

func TestActivityWaitPermanentReadErrorsAreNotRetried(t *testing.T) {
	permanents := []error{
		StatusError{Code: http.StatusBadRequest},
		StatusError{Code: http.StatusUnauthorized},
		StatusError{Code: http.StatusForbidden},
		errors.New("json: cannot unmarshal string into Go value"),
	}
	for _, permanent := range permanents {
		t.Run(fmt.Sprintf("%v", permanent), func(t *testing.T) {
			calls := 0
			read := scriptedReads(&calls, readOutcome{err: permanent})
			_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
			if err == nil {
				t.Fatal("permanent errors must fail")
			}
			if calls != 1 {
				t.Fatalf("calls=%d, want exactly 1 (no retry on permanent errors)", calls)
			}
		})
	}
}

func TestActivityWaitTerminalFailureIsNotRetried(t *testing.T) {
	calls := 0
	read := scriptedReads(&calls, readOutcome{activity: failedActivity()})
	_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
	if err == nil {
		t.Fatal("a failed activity must surface an error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want exactly 1 (failed is terminal, retrying would hide real platform failures)", calls)
	}
}

func TestActivityWaitFirstNotFoundIsTolerated(t *testing.T) {
	t.Run("initial not-found then completed succeeds", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls,
			readOutcome{}, // nil activity, nil error: not found
			readOutcome{activity: completedActivity()},
		)
		activity, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
		if err != nil || activity == nil {
			t.Fatalf("eventual consistency on the first read must be tolerated, got err=%v", err)
		}
	})

	t.Run("repeated not-found is permanent", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls, readOutcome{}, readOutcome{})
		_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
		if err == nil {
			t.Fatal("a repeatedly missing activity must fail")
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want exactly 2", calls)
		}
	})
}

func TestActivityWaitPollingIsBoundedByInjectedBackoff(t *testing.T) {
	calls := 0
	read := scriptedReads(&calls, readOutcome{activity: pendingActivity()})
	_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(3), nil)
	if err == nil {
		t.Fatal("an activity stuck in a non-terminal state must be bounded by the backoff/context")
	}
	if calls != 4 {
		t.Fatalf("calls=%d, want 4 (initial + 3 retries)", calls)
	}
}

func TestActivityWaitContextCancellationStopsPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	read := func(c context.Context) (*Activity, error) {
		calls++
		cancel()
		return pendingActivity(), nil
	}
	_, err := waitForActivityCompletion(ctx, "act-1", read, immediateBackoff(50), nil)
	if err == nil {
		t.Fatal("context cancellation must stop the polling")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 (no poll after cancellation)", calls)
	}
}

func TestIsTransientAPIError(t *testing.T) {
	if isTransientAPIError(StatusError{Code: http.StatusNotFound}) {
		t.Fatal("404 is handled by the not-found path, not the transient classifier")
	}
	if isTransientAPIError(errors.New("decode error")) {
		t.Fatal("arbitrary errors are permanent")
	}
	if !isTransientAPIError(fmt.Errorf("wrapped: %w", StatusError{Code: http.StatusBadGateway})) {
		t.Fatal("wrapped transient StatusError must stay transient")
	}
}

func TestIsTransientActivityFailure(t *testing.T) {
	transient := &ActivityCompletionError{
		activity: &Activity{State: map[string]ActivityState{
			"failed": {Reason: "None of the workers were able to respond in time"},
		}},
	}
	if !IsTransientActivityFailure(transient) {
		t.Fatal("the known transient platform reason must be retryable")
	}
	if !IsTransientActivityFailure(fmt.Errorf("wrapped: %w", transient)) {
		t.Fatal("a wrapped transient failure must stay transient")
	}

	permanent := &ActivityCompletionError{
		activity: &Activity{State: map[string]ActivityState{
			"failed": {Reason: "MAC address is already used by virtual machine vm-2"},
		}},
	}
	if IsTransientActivityFailure(permanent) {
		t.Fatal("an arbitrary failure reason must NOT be retryable (narrow matcher)")
	}
	if IsTransientActivityFailure(&ActivityCompletionError{}) {
		t.Fatal("a completion error without activity must not be retryable")
	}
	if IsTransientActivityFailure(errors.New("plain error")) {
		t.Fatal("a non-activity error must not be retryable")
	}
}
