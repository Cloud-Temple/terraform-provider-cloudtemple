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
	if !isTransientAPIError(&url.Error{Op: "Get", URL: "https://shiva.example", Err: errors.New("connection reset")}) {
		t.Fatal("a non-timeout transport error (connection reset) must stay transient")
	}
	// A CONFIGURED request timeout (or ctx deadline/cancel) must NOT be transient:
	// retrying it in a waiter would multiply the per-request bound into a
	// multi-minute stall (issue #339).
	if isTransientAPIError(&url.Error{Op: "Get", URL: "https://shiva.example", Err: timeoutErr{}}) {
		t.Fatal("a configured request timeout must NOT be transient")
	}
	if isTransientAPIError(context.DeadlineExceeded) {
		t.Fatal("a context deadline must not be transient")
	}
	if isTransientAPIError(context.Canceled) {
		t.Fatal("a context cancellation must not be transient")
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

	// VPC transient gateway hiccup (#315/#319): the write activity fails with
	// "Failed to load configuration via API: …502 Bad Gateway…nginx…". The lead
	// phrase requires a 502 / Bad Gateway corroboration (AND-match): the matcher
	// is GLOBAL (it also gates VIF/compute retries), so the lead phrase alone is
	// too broad — a genuine PERMANENT config-load failure could carry it. A bare
	// 502 without the lead phrase is also rejected: a non-idempotent upstream 502
	// must stay fatal. Fail-closed (a false negative beats a false positive).
	vpcFailure := func(reason string) *ActivityCompletionError {
		return &ActivityCompletionError{activity: &Activity{State: map[string]ActivityState{
			"failed": {Reason: reason},
		}}}
	}
	if !IsTransientActivityFailure(vpcFailure("Failed to load configuration via API: <html><body><h1>502 Bad Gateway</h1></body></html> nginx/1.22.1")) {
		t.Fatal("the VPC transient 502 (lead phrase + 502 + Bad Gateway) must be retryable")
	}
	if !IsTransientActivityFailure(vpcFailure("Failed to load configuration via API: 502 upstream error")) {
		t.Fatal("the lead phrase corroborated by 502 alone must be retryable")
	}
	if !IsTransientActivityFailure(vpcFailure("Failed to load configuration via API: Bad Gateway from upstream")) {
		t.Fatal("the lead phrase corroborated by Bad Gateway alone must be retryable")
	}
	if IsTransientActivityFailure(vpcFailure("Failed to load configuration via API: invalid schema definition")) {
		t.Fatal("the lead phrase WITHOUT a 502/Bad Gateway corroboration must NOT be retryable (a permanent config-load failure stays fatal); kills the marker-alone / OR mutant")
	}
	if IsTransientActivityFailure(vpcFailure("upstream returned 502 Bad Gateway")) {
		t.Fatal("a bare 502/Bad Gateway WITHOUT the lead phrase must NOT be retryable (too broad); kills the broaden-to-502 mutant")
	}
}

func TestActivityWaitNotFoundToleranceSurvivesTransientBlips(t *testing.T) {
	// FF-3 finding: the one-time not-found tolerance was consumed by any
	// prior attempt (total counter), so a transient blip BEFORE the
	// activity became visible turned the first legitimate not-found into
	// a permanent failure.
	calls := 0
	read := scriptedReads(&calls,
		readOutcome{err: StatusError{Code: http.StatusInternalServerError}},
		readOutcome{}, // first not-found: must be tolerated once
		readOutcome{activity: completedActivity()},
	)
	activity, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
	if err != nil || activity == nil {
		t.Fatalf("a transient blip before the first not-found must not consume the tolerance, got err=%v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d, want 3", calls)
	}
}

func TestActivityWaitDisappearedActivityIsPermanent(t *testing.T) {
	calls := 0
	read := scriptedReads(&calls,
		readOutcome{activity: pendingActivity()},
		readOutcome{}, // the activity was seen, then vanished: permanent
	)
	_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), nil)
	if err == nil {
		t.Fatal("an activity that was seen and vanished must fail permanently")
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2 (no retry after disappearance)", calls)
	}
}

// TestActivityWaitNotFoundBudgetIsConfigurableAndBounded pins the E0-7
// anti-orphan extension (#415): WaiterOptions.NotFoundRetries widens the
// initial not-found tolerance from a single read to a bounded budget, so a
// slow-to-index activity does not fail a write that is still running
// platform-side. The default (unset) budget stays at one, and the extension
// never relaxes the disappearance rule.
func TestActivityWaitNotFoundBudgetIsConfigurableAndBounded(t *testing.T) {
	t.Run("a generous budget tolerates several initial not-founds then completes", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls,
			readOutcome{}, // three initial not-founds (eventual consistency)
			readOutcome{},
			readOutcome{},
			readOutcome{activity: completedActivity()},
		)
		opts := &WaiterOptions{NotFoundRetries: 5}
		activity, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), opts)
		if err != nil || activity == nil {
			t.Fatalf("a budget of 5 must tolerate 3 initial not-founds, got err=%v", err)
		}
		if calls != 4 {
			t.Fatalf("calls=%d, want 4", calls)
		}
	})

	t.Run("the budget is bounded: a not-found beyond it is permanent", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls, readOutcome{}) // endless not-found
		opts := &WaiterOptions{NotFoundRetries: 3}
		_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(50), opts)
		if err == nil {
			t.Fatal("an endless not-found must fail once the budget is exhausted")
		}
		// 3 tolerated retries + 1 final permanent attempt; kills the
		// unbounded-tolerance and off-by-one mutants.
		if calls != 4 {
			t.Fatalf("calls=%d, want 4 (3 tolerated + 1 permanent)", calls)
		}
	})

	t.Run("a generous budget still treats a seen-then-vanished activity as permanent", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls,
			readOutcome{activity: pendingActivity()}, // seen
			readOutcome{},                            // vanished
		)
		opts := &WaiterOptions{NotFoundRetries: 10}
		_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), opts)
		if err == nil {
			t.Fatal("disappearance after being seen is permanent regardless of the not-found budget")
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2 (disappearance is not covered by the not-found budget)", calls)
		}
	})

	t.Run("the default (unset) budget still tolerates exactly one initial not-found", func(t *testing.T) {
		calls := 0
		read := scriptedReads(&calls, readOutcome{}, readOutcome{})
		opts := &WaiterOptions{} // NotFoundRetries unset -> historical single tolerance
		_, err := waitForActivityCompletion(context.Background(), "act-1", read, immediateBackoff(20), opts)
		if err == nil {
			t.Fatal("the default budget tolerates only one initial not-found")
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2 (default budget unchanged)", calls)
		}
	})
}
