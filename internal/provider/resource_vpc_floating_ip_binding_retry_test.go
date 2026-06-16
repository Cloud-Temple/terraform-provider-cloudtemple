package provider

import (
	"context"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// These tests pin the bounded retry on the transient platform gateway 502
// (#315/#319) wired into bind/unbind. The transient activity failure is
// simulated with the errVPCTransient sentinel + the vpcIsTransient classifier
// (both from vpc_write_retry_test.go), because the provider package cannot
// build a real client.ActivityCompletionError (its fields are unexported).
//
// The overriding invariant: a retry NEVER clobbers a binding that changed
// between attempts, and NEVER re-issues a write that already converged.

func TestCreateVPCFloatingIPBindingRetriesTransient502(t *testing.T) {
	ctx := context.Background()
	inconclusiveCorroborate := func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
		return client.FloatingIPBindingInconclusive, nil
	}

	t.Run("transient wait then UNBOUND on re-read -> re-bind, converges", func(t *testing.T) {
		// Pre-bind shows the FIP free -> bind proceeds. Attempt 1's wait fails
		// transient; attempt 2 re-reads UNBOUND -> re-bind, wait OK; the
		// confirmation read then shows our pair.
		d := emptyBindingState(t)
		var bindCalls, waitCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind: free
				readResult{unboundFIP(), nil}, // retry re-classify: still free -> re-bind
				readResult{boundFIP(), nil},   // confirmation: converged
			),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: func(ctx context.Context, activityID string) error {
				waitCalls++
				if waitCalls == 1 {
					return errVPCTransient
				}
				return nil
			},
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a transient wait failure must be retried to convergence, got: %v", diags)
		}
		if bindCalls != 2 {
			t.Fatalf("expected 2 bind POSTs (initial + one retry while still unbound), got %d", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set after convergence, got %q", d.Id())
		}
	})

	t.Run("transient wait then BOUND-TO-TARGET on re-read -> success, NO second bind POST", func(t *testing.T) {
		// The bind converged platform-side despite the transient wait failure:
		// the retry re-reads BoundToTarget and must SUCCEED without re-binding.
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind: free
				readResult{boundFIP(), nil},   // retry re-classify: converged platform-side
				readResult{boundFIP(), nil},   // confirmation
			),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errVPCTransient }, // every wait fails
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a bind that converged platform-side after a transient failure must succeed, got: %v", diags)
		}
		if bindCalls != 1 {
			t.Fatalf("a converged bind must NOT be re-issued on retry, got %d bind POSTs; kills the re-bind-on-converged mutant", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set after adopting the converged bind, got %q", d.Id())
		}
	})

	t.Run("transient wait then BOUND-TO-OTHER on re-read -> anti-clobber abort, NO second bind POST", func(t *testing.T) {
		// Between retries the FIP becomes bound to a DIFFERENT static IP: the
		// retry must ABORT, never re-bind (anti-clobber).
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil},    // pre-bind: free
				readResult{otherBoundFIP(), nil}, // retry re-classify: clobber risk
			),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errVPCTransient },
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a FIP that became bound to a different static IP between retries must ABORT (anti-clobber)")
		}
		if bindCalls != 1 {
			t.Fatalf("the retry must NOT re-bind a FIP bound elsewhere, got %d bind POSTs; kills the re-bind-on-bound-to-other mutant", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("an aborted create must not set the id, got %q", d.Id())
		}
	})

	t.Run("persistent transient wait -> bounded, fails without converging", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			// Pre-bind free, and every retry re-read still free -> re-bind each time.
			read:        readSeq(&readCalls, readResult{unboundFIP(), nil}),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errVPCTransient },
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a persistent transient failure must eventually fail (bounded)")
		}
		if bindCalls != maxTransientVPCAttempts {
			t.Fatalf("expected exactly %d bind POSTs (bounded budget), got %d", maxTransientVPCAttempts, bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a failed create must not set the id, got %q", d.Id())
		}
	})

	t.Run("non-transient wait failure is NOT retried", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errVPCPermanent },
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a non-transient wait failure must fail the create")
		}
		if bindCalls != 1 {
			t.Fatalf("a non-transient failure must NOT be retried, got %d bind POSTs", bindCalls)
		}
	})

	t.Run("transient wait then AMBIGUOUS re-read corroborated UNBOUND -> re-bind, converges", func(t *testing.T) {
		// The retry's per-id re-read is ambiguous (nil/403) and must fall back to
		// the strict listing (corroborate). A corroborated UNBOUND unlocks a re-bind.
		d := emptyBindingState(t)
		var bindCalls, waitCalls, readCalls, corrCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind: free
				readResult{nil, nil},          // retry re-classify: ambiguous -> corroborate
				readResult{boundFIP(), nil},   // confirmation
			),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: func(ctx context.Context, activityID string) error {
				waitCalls++
				if waitCalls == 1 {
					return errVPCTransient
				}
				return nil
			},
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				corrCalls++
				return client.FloatingIPBindingUnbound, nil
			},
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("an ambiguous re-read corroborated UNBOUND must allow a re-bind, got: %v", diags)
		}
		if corrCalls != 1 {
			t.Fatalf("the retry must fall back to the strict listing on an ambiguous re-read, got %d corroborate calls", corrCalls)
		}
		if bindCalls != 2 {
			t.Fatalf("a corroborated-unbound retry must re-bind, got %d bind POSTs", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set after convergence, got %q", d.Id())
		}
	})

	t.Run("transient wait then AMBIGUOUS re-read corroborated BOUND-TO-OTHER -> abort, NO second bind POST", func(t *testing.T) {
		// The retry's per-id re-read is ambiguous; the strict listing shows the FIP
		// bound elsewhere -> anti-clobber abort, never re-bind on ambiguous evidence.
		d := emptyBindingState(t)
		var bindCalls, readCalls, corrCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind: free
				readResult{nil, nil},          // retry re-classify: ambiguous -> corroborate
			),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: func(ctx context.Context, activityID string) error { return errVPCTransient },
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				corrCalls++
				return client.FloatingIPBindingBoundToOther, nil
			},
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an ambiguous re-read corroborated BOUND-TO-OTHER must ABORT (anti-clobber)")
		}
		if corrCalls != 1 {
			t.Fatalf("the retry must corroborate on an ambiguous re-read, got %d corroborate calls", corrCalls)
		}
		if bindCalls != 1 {
			t.Fatalf("a corroborated-bound-elsewhere retry must NOT re-bind, got %d bind POSTs", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("an aborted create must not set the id, got %q", d.Id())
		}
	})

	t.Run("transient wait then AMBIGUOUS re-read corroborated INCONCLUSIVE -> abort, NO second bind POST", func(t *testing.T) {
		// The retry's per-id re-read is ambiguous; the strict listing is itself
		// Inconclusive -> fail-closed abort, never re-bind on ambiguous evidence
		// (the dedicated Inconclusive branch, distinct from BoundToOther).
		d := emptyBindingState(t)
		var bindCalls, readCalls, corrCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind: free
				readResult{nil, nil},          // retry re-classify: ambiguous -> corroborate
			),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: func(ctx context.Context, activityID string) error { return errVPCTransient },
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				corrCalls++
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an ambiguous re-read corroborated INCONCLUSIVE must FAIL CLOSED (no re-bind on ambiguous evidence)")
		}
		if corrCalls != 1 {
			t.Fatalf("the retry must corroborate on an ambiguous re-read, got %d corroborate calls", corrCalls)
		}
		if bindCalls != 1 {
			t.Fatalf("a corroborated-inconclusive retry must NOT re-bind, got %d bind POSTs; kills the Inconclusive-falls-through-to-rebind mutant", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a fail-closed create must not set the id, got %q", d.Id())
		}
	})
}

func TestDeleteVPCFloatingIPBindingRetriesTransient502(t *testing.T) {
	ctx := context.Background()
	inconclusiveCorroborate := func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
		return client.FloatingIPBindingInconclusive, nil
	}

	t.Run("transient wait then UNBOUND on re-read -> success, NO second unbind", func(t *testing.T) {
		// The unbind converged platform-side despite the transient wait failure:
		// the retry re-classifies Unbound and must SUCCEED without re-unbinding.
		d := bindingState(t)
		var unbindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // retry re-classify: no longer our pair
				readResult{unboundFIP(), nil}, // external confirm: unbound
			),
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { unbindCalls++; return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errVPCTransient }, // attempt 1 wait fails
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("an unbind that converged platform-side after a transient failure must succeed, got: %v", diags)
		}
		if unbindCalls != 1 {
			t.Fatalf("a converged unbind must NOT be re-issued on retry, got %d unbind calls; kills the re-unbind-on-converged mutant", unbindCalls)
		}
	})

	t.Run("transient wait then STILL-BOUND on re-read -> re-unbind, converges", func(t *testing.T) {
		d := bindingState(t)
		var unbindCalls, waitCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: readSeq(&readCalls,
				readResult{boundFIP(), nil},   // retry re-classify: still our pair
				readResult{unboundFIP(), nil}, // external confirm: unbound after the re-unbind
			),
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { unbindCalls++; return "act", nil },
			wait: func(ctx context.Context, activityID string) error {
				waitCalls++
				if waitCalls == 1 {
					return errVPCTransient
				}
				return nil
			},
			corroborate: inconclusiveCorroborate,
			sleep:       noSleep,
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a still-bound pair must be re-unbound to convergence, got: %v", diags)
		}
		if unbindCalls != 2 {
			t.Fatalf("expected 2 unbind calls (initial + one retry while still bound), got %d", unbindCalls)
		}
	})
}
