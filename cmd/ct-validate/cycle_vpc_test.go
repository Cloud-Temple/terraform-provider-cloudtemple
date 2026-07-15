package main

import (
	"context"
	"net/http"
	"testing"
)

// TestVPCExplicitDeleteStaticIPMarksProofOnlyWhenClosureRuns is the VPC analog of
// TestExplicitDeleteMarksProofOnlyWhenClosureRuns (compute). The same-cycle delete
// proof (ref.ExplicitlyDeleted) is what later lets the deferred teardown accept a
// 403-on-absent as "confirmed deleted" (channel 1 in staticIPDeleteErrResult)
// instead of falling back to the strict by-network listing (channel 2). It MUST be
// set from INSIDE the breaker-gated r.op closure, on a delete that actually RAN and
// fully succeeded — never inferred from r.op's return value, which is nil on a
// breaker SKIP too.
//
// If the proof were (wrongly) set on a skip, a later 403-on-absent would be masked
// as "already deleted" and a still-present forbidden static IP would orphan
// silently — the exact never-orphan failure the harness exists to prevent.
//
// Mutations this test turns RED:
//   - set ref.ExplicitlyDeleted from `r.op(...) == nil` instead of inside the
//     closure → case (a) marks the proof true on a skip → RED.
//   - set the proof before the `if derr != nil` check → case (c) marks it on a
//     failed delete → RED.
func TestVPCExplicitDeleteStaticIPMarksProofOnlyWhenClosureRuns(t *testing.T) {
	ctx := context.Background()

	// (a) breaker tripped → op skipped → closure NOT run → proof stays false. The
	// closure holds the ONLY DELETE, so a skip must issue zero HTTP: the stub fails
	// the test if it receives any request.
	t.Run("skip_does_not_set_proof", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("a breaker-skipped delete must issue NO request, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		})
		b := NewBreaker(1000, 0.99, 1000)
		b.Trip("force-skip")
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		ref := &staticIPTeardownRef{ID: "si-1"}

		vpcCycle{}.explicitDeleteStaticIP(ctx, c, r, ref)

		if ref.ExplicitlyDeleted {
			t.Fatal("a breaker-skipped (never-run) delete must NOT set the same-cycle proof (else a later 403 masks an orphan)")
		}
		if op, ok := opsByEndpoint(r)["vpc.static_ip.delete"]; !ok || !op.Skipped {
			t.Fatalf("expected vpc.static_ip.delete recorded as a skip, got %+v (present=%v)", op, ok)
		}
	})

	// (b) breaker allows + delete fully succeeds: the async DELETE returns 200 +
	// Location, then the delete activity reports "completed" → proof set.
	t.Run("success_sets_proof", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/static_ips/si-1":
				w.Header().Set("Location", "act-del")
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-del":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"act-del","state":{"completed":{"result":"si-1"}}}`))
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
		ref := &staticIPTeardownRef{ID: "si-1"}

		vpcCycle{}.explicitDeleteStaticIP(ctx, c, r, ref)

		if !ref.ExplicitlyDeleted {
			t.Fatal("a ran-and-succeeded explicit delete must set the same-cycle proof")
		}
	})

	// (c) breaker allows + delete fails (DELETE 500) → the closure returns the error
	// before reaching the proof assignment → proof NOT set.
	t.Run("failure_does_not_set_proof", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/static_ips/si-1":
				w.WriteHeader(http.StatusInternalServerError)
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
		ref := &staticIPTeardownRef{ID: "si-1"}

		vpcCycle{}.explicitDeleteStaticIP(ctx, c, r, ref)

		if ref.ExplicitlyDeleted {
			t.Fatal("a failed explicit delete must NOT set the same-cycle proof")
		}
	})
}

// TestVPCExplicitDeprovisionFloatingIPMarksProofOnlyWhenClosureRuns is the FIP
// analog of the static-IP proof test, but the proof's ROLE differs and the test is
// honest about that. Unlike the static ref — where ExplicitlyDeleted GATES whether
// a later 403-on-absent is accepted as "deleted" (a correctness gate) — the FIP
// teardown idempotency does NOT depend on ref.ExplicitlyDeleted at all: the FIP
// by-id read returns an authoritative 404 once gone and DeprovisionUnbound maps
// that 404 to success (see floatingIPTeardownRef doc). So for the FIP the proof is
// DIAGNOSTICS + PARITY, not a safety gate.
//
// The proof-setting DISCIPLINE still matters, which is what this test pins: the
// same-cycle proof is the postmortem signal a human reads during the §6 live run to
// answer "did the cycle actually deprovision this BILLABLE IP, or did it leak?".
// If the proof were (wrongly) set on a breaker SKIP or on a FAILED deprovision, that
// audit signal would falsely report a release that never happened — exactly the kind
// of misleading "all clear" that hides a billable orphan from the human reviewer.
//
// Mutations this test turns RED:
//   - set ref.ExplicitlyDeleted from `r.op(...) == nil` instead of inside the
//     closure → case (a) marks the proof true on a skip → RED.
//   - move the proof assignment before the `if derr != nil` check → case (c) marks
//     it on a failed deprovision → RED.
func TestVPCExplicitDeprovisionFloatingIPMarksProofOnlyWhenClosureRuns(t *testing.T) {
	ctx := context.Background()

	// (a) breaker tripped → op skipped → closure NOT run → proof stays false. The
	// closure holds the ONLY deprovision (resolve → DELETE → confirm), so a skip must
	// issue zero HTTP: the stub fails the test on any request.
	t.Run("skip_does_not_set_proof", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("a breaker-skipped deprovision must issue NO request, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		})
		b := NewBreaker(1000, 0.99, 1000)
		b.Trip("force-skip")
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		ref := &floatingIPTeardownRef{ID: "fip-1"}

		vpcCycle{}.explicitDeprovisionFloatingIP(ctx, c, r, ref)

		if ref.ExplicitlyDeleted {
			t.Fatal("a breaker-skipped (never-run) deprovision must NOT set the same-cycle proof (else the §6 audit falsely reports a release)")
		}
		if op, ok := opsByEndpoint(r)["vpc.floating_ip.deprovision"]; !ok || !op.Skipped {
			t.Fatalf("expected vpc.floating_ip.deprovision recorded as a skip, got %+v (present=%v)", op, ok)
		}
	})

	// (b) breaker allows + deprovision fully succeeds. The full gated-and-confirmed
	// path runs: resolve #1 (200, fully unbound) → DELETE (200 + Location) → activity
	// completes → resolve #2 (404, positive absence confirm). A getCount discriminates
	// the two by-id reads: the first must observe an unbound FIP, the second a 404.
	t.Run("success_sets_proof", func(t *testing.T) {
		getCount := 0
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				getCount++
				if getCount == 1 {
					// Resolve #1: present and PROVABLY FULLY UNBOUND. The gate requires all
					// three associations PRESENT-AND-EXPLICITLY-NULL in the raw body (an
					// omitted field is not proof, #310), so the body must spell them out.
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
					return
				}
				// Resolve #2: authoritative 404 → positive absence confirm.
				w.WriteHeader(http.StatusNotFound)
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.Header().Set("Location", "act-deprov")
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-deprov":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"act-deprov","state":{"completed":{"result":"fip-1"}}}`))
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
		ref := &floatingIPTeardownRef{ID: "fip-1"}

		vpcCycle{}.explicitDeprovisionFloatingIP(ctx, c, r, ref)

		if !ref.ExplicitlyDeleted {
			t.Fatal("a ran-and-succeeded explicit deprovision must set the same-cycle proof")
		}
		if getCount != 2 {
			t.Fatalf("expected exactly 2 by-id reads (resolve + positive-404 confirm), got %d", getCount)
		}
	})

	// (c) breaker allows + deprovision fails. The FIP is observed unbound (resolve #1
	// 200), so the gate opens and the DELETE runs — but the DELETE returns 500, so
	// DeprovisionUnbound errors and the closure returns before the proof assignment.
	// This mirrors the static failure case (a delete that RAN but failed) and proves
	// the proof is never set on a failed release.
	t.Run("failure_does_not_set_proof", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				// Provably fully unbound so the gate OPENS and the DELETE actually runs;
				// the failure must come from the DELETE (500), not from a gate refusal,
				// otherwise this would assert the wrong path (#310 present-and-null proof).
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusInternalServerError)
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
		ref := &floatingIPTeardownRef{ID: "fip-1"}

		vpcCycle{}.explicitDeprovisionFloatingIP(ctx, c, r, ref)

		if ref.ExplicitlyDeleted {
			t.Fatal("a failed explicit deprovision must NOT set the same-cycle proof")
		}
	})
}
