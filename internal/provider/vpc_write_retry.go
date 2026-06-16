package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// maxTransientVPCAttempts bounds the TOTAL number of attempts (not retries) of
// a VPC async write whose activity failed for a transient platform reason (the
// gateway 502 "Failed to load configuration via API: …502 Bad Gateway…",
// #315/#319). Parity with the VIF retry budget (#251, maxTransientVIFAttempts).
const maxTransientVPCAttempts = 3

// vpcWriteRetry holds the injectable seams of the bounded VPC write retry so
// the loop is unit tested without HTTP calls or sleeps.
//
// The helper is deliberately MINIMAL: it owns only the retry MECHANICS
// (attempt budget, backoff, context, the transient gate, the last-error
// propagation, the log line). It knows NOTHING about static IPs, floating IPs,
// binding classification or deletion confirmation. Every per-operation decision
// — re-read the live resource, re-classify safety/convergence, decide whether
// to re-emit the write, and (crucially) run the source-of-truth confirmations —
// stays in the caller's `attempt` closure. That separation keeps the anti-clobber
// and never-orphan invariants (#275/#281/#303/#312) out of this generic loop.
type vpcWriteRetry struct {
	// attempt performs ONE full try and returns nil on success/convergence,
	// or the error to be classified. The caller's closure does, on EVERY call:
	// (1) re-read the live resource, (2) re-classify safety/convergence (e.g.
	// already-converged → return nil without re-emitting; bound-to-other →
	// return a non-transient abort error), (3) if still required, re-emit the
	// write, (4) wait for the activity. Only a transient activity failure
	// (isTransient) is retried; anything else returns immediately.
	attempt func(ctx context.Context) error
	// sleep waits between attempts; it returns ctx.Err() on cancellation.
	// Defaults to defaultVPCSleep (attempt*10s) when nil.
	sleep func(ctx context.Context, attempt int) error
	// isTransient classifies a failure as retryable; defaults to
	// client.IsTransientActivityFailure (injectable for tests).
	isTransient func(err error) bool
	// label names the operation in the retry log line and wrapped errors.
	label string
}

// defaultVPCSleep backs off attempt*10s between attempts, respecting ctx.
// Mirrors defaultVIFSleep so the two retry paths behave identically.
func defaultVPCSleep(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt) * 10 * time.Second):
		return nil
	}
}

// runVPCWriteWithRetry drives r.attempt with a bounded retry on transient
// platform activity failures. Invariants defended by the unit tests:
//   - at most maxTransientVPCAttempts TOTAL attempts;
//   - only an isTransient failure is retried; any other error (a sync POST
//     StatusError, a non-transient activity failure, a caller abort such as
//     bind-to-other) returns immediately, never retried;
//   - no extra attempt after a success (attempt returns nil);
//   - the context is honoured at the head of every attempt AND while sleeping;
//   - on budget exhaustion the last transient error is returned (fail-closed).
//
// The total wall-clock is NOT bounded by the per-request HTTP timeout: it can
// span up to maxTransientVPCAttempts full platform activities plus the
// attempt*10s backoffs and each activity's internal polling. It is bounded by
// the Terraform operation context — callers rely on that ctx, not on a fixed
// number of polls (#319 Codex PLAN, budget note).
func runVPCWriteWithRetry(ctx context.Context, r vpcWriteRetry) error {
	if r.sleep == nil {
		r.sleep = defaultVPCSleep
	}
	if r.isTransient == nil {
		r.isTransient = client.IsTransientActivityFailure
	}

	var lastErr error
	for attempt := 1; attempt <= maxTransientVPCAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if lastErr != nil {
				return fmt.Errorf("cancelled while retrying %s: %w (last transient failure: %s)", r.label, ctxErr, lastErr)
			}
			return ctxErr
		}

		err := r.attempt(ctx)
		if err == nil {
			return nil
		}
		if !r.isTransient(err) {
			return err
		}
		lastErr = err

		if attempt == maxTransientVPCAttempts {
			break
		}
		tflog.Warn(ctx, fmt.Sprintf("%s: transient platform failure (attempt %d/%d), retrying: %s",
			r.label, attempt, maxTransientVPCAttempts, err))
		if sleepErr := r.sleep(ctx, attempt); sleepErr != nil {
			return fmt.Errorf("cancelled while retrying %s: %w (last transient failure: %s)", r.label, sleepErr, lastErr)
		}
	}
	return lastErr
}
