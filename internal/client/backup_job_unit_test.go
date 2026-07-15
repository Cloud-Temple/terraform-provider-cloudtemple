package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// bjOutcome scripts one read of the backup-job waiter.
type bjOutcome struct {
	job *BackupJob
	err error
}

// scriptedBackupJobReads replays the given outcomes in order and counts calls;
// the last outcome repeats if the loop polls beyond the script.
func scriptedBackupJobReads(calls *int, outcomes ...bjOutcome) backupJobReadFunc {
	return func(ctx context.Context) (*BackupJob, error) {
		i := *calls
		if i >= len(outcomes) {
			i = len(outcomes) - 1
		}
		*calls++
		return outcomes[i].job, outcomes[i].err
	}
}

func idleJob() *BackupJob    { return &BackupJob{ID: "job-1", Status: "IDLE"} }
func runningJob() *BackupJob { return &BackupJob{ID: "job-1", Status: "RUNNING"} }

func TestWaitForBackupJobTransientReadErrorsAreRetried(t *testing.T) {
	calls := 0
	read := scriptedBackupJobReads(&calls,
		bjOutcome{err: StatusError{Code: http.StatusInternalServerError}},
		bjOutcome{err: StatusError{Code: http.StatusBadGateway}},
		bjOutcome{job: idleJob()},
	)
	job, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
	if err != nil || job == nil {
		t.Fatalf("transient read errors must be survived then IDLE succeeds, got err=%v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d, want 3 (two transient + IDLE)", calls)
	}
}

// TestWaitForBackupJobNonRetryableReadErrorsFailAtOnce covers permanent errors
// AND configured timeout / context errors: the waiter doc (backup_job.go) requires
// a configured request timeout or a context error to fail at once, never retried
// (retrying a timeout would stall for minutes).
func TestWaitForBackupJobNonRetryableReadErrorsFailAtOnce(t *testing.T) {
	nonRetryable := []error{
		StatusError{Code: http.StatusBadRequest},
		errors.New("json: cannot unmarshal string into Go value"),
		context.Canceled,
		context.DeadlineExceeded,
		&url.Error{Op: "Get", URL: "https://shiva.example", Err: timeoutErr{}},
	}
	for _, e := range nonRetryable {
		t.Run(fmt.Sprintf("%v", e), func(t *testing.T) {
			calls := 0
			read := scriptedBackupJobReads(&calls, bjOutcome{err: e})
			_, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
			if err == nil {
				t.Fatal("a non-retryable read error must fail the wait")
			}
			if calls != 1 {
				t.Fatalf("calls=%d, want exactly 1 (no retry on a permanent/timeout/ctx error)", calls)
			}
		})
	}
}

// TestWaitForBackupJobNotFoundToleranceIsFirstAttemptOnly pins the CURRENT
// count==1 semantics (#293 Finding F2): the not-found tolerance is "nil job on the
// very FIRST polling attempt", not "first nil occurrence". A prior attempt — here a
// transient blip — consumes the tolerance, so a subsequent nil job is immediately
// fatal. T6(b) is the mutation-killer that forbids "fixing" this into activity-style
// notFoundTolerated/activitySeen (FF-3) under this test PR.
func TestWaitForBackupJobNotFoundToleranceIsFirstAttemptOnly(t *testing.T) {
	t.Run("nil job on the first attempt then IDLE succeeds", func(t *testing.T) {
		calls := 0
		read := scriptedBackupJobReads(&calls,
			bjOutcome{}, // nil job, nil err: not found on attempt 1 (tolerated)
			bjOutcome{job: idleJob()},
		)
		job, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
		if err != nil || job == nil {
			t.Fatalf("a first-attempt not-found must be tolerated, got err=%v", err)
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2", calls)
		}
	})

	t.Run("a transient blip before the nil job consumes the count==1 tolerance => fatal", func(t *testing.T) {
		calls := 0
		read := scriptedBackupJobReads(&calls,
			bjOutcome{err: StatusError{Code: http.StatusInternalServerError}}, // attempt 1 (count==1) consumed here
			bjOutcome{}, // attempt 2: nil job, count!=1 => permanent
		)
		_, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
		if err == nil {
			t.Fatal("a nil job after the count==1 tolerance was consumed must be fatal (pins current behavior, F2)")
		}
		if calls != 2 {
			t.Fatalf("calls=%d, want 2 (the prior transient attempt consumed the tolerance)", calls)
		}
	})
}

func TestWaitForBackupJobRunningKeepsPolling(t *testing.T) {
	calls := 0
	read := scriptedBackupJobReads(&calls,
		bjOutcome{job: runningJob()},
		bjOutcome{job: idleJob()},
	)
	job, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
	if err != nil || job == nil {
		t.Fatalf("RUNNING must keep polling then IDLE succeeds, got err=%v", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2 (RUNNING is not terminal)", calls)
	}
}

func TestWaitForBackupJobUnknownStatusIsTerminal(t *testing.T) {
	calls := 0
	read := scriptedBackupJobReads(&calls, bjOutcome{job: &BackupJob{ID: "job-1", Status: "FAILED"}})
	_, err := waitForBackupJobCompletion(context.Background(), "job-1", read, immediateBackoff(20), nil)
	if err == nil {
		t.Fatal("a non-IDLE/non-RUNNING status must fail without retry")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want exactly 1 (unknown status is terminal)", calls)
	}
}

func TestWaitForBackupJobContextCancellationStopsPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	read := func(c context.Context) (*BackupJob, error) {
		calls++
		cancel()
		return runningJob(), nil
	}
	_, err := waitForBackupJobCompletion(ctx, "job-1", read, immediateBackoff(50), nil)
	if err == nil {
		t.Fatal("context cancellation must stop the polling")
	}
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 (the loop must honour the retry ctx, not context.Background())", calls)
	}
}
