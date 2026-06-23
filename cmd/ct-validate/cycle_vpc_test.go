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
