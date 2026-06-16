package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// These tests pin the bounded retry on the transient platform gateway 502
// (#315/#319) wired into the static IP delete and update. The transient
// activity failure is simulated with errVPCTransient + vpcIsTransient (from
// vpc_write_retry_test.go), since the provider package cannot build a real
// client.ActivityCompletionError.

func siDescPtr(s string) *string { return &s }

// delSeq scripts a delete func: yields the supplied (activityID, err) outcomes
// in order, repeating the last; counts the calls.
func delSeq(calls *int, results ...struct {
	id  string
	err error
}) vpcStaticIPDeleteFunc {
	return func(ctx context.Context, id string) (string, error) {
		i := *calls
		*calls++
		if i >= len(results) {
			i = len(results) - 1
		}
		return results[i].id, results[i].err
	}
}

type delResult = struct {
	id  string
	err error
}

func TestDeleteVPCStaticIPRetriesTransient502(t *testing.T) {
	ctx := context.Background()
	noList := siListStrict()

	t.Run("transient wait then re-DELETE 404 -> idempotent success", func(t *testing.T) {
		// Attempt 1's DELETE starts an activity whose wait fails transient; the
		// delete actually took effect, so attempt 2's re-DELETE returns 404 ->
		// idempotent success.
		d := newStaticIPState(t)
		var delCalls, waitCalls int
		del := delSeq(&delCalls,
			delResult{"act", nil},
			delResult{"", client.StatusError{Code: http.StatusNotFound}},
		)
		wait := func(ctx context.Context, activityID string) error { waitCalls++; return errVPCTransient }
		diags := deleteVPCStaticIPWithRetry(ctx, d, del, noList, wait, noSleep, vpcIsTransient)
		if diags.HasError() {
			t.Fatalf("a transient delete must be retried; a re-DELETE 404 proves it converged, got: %v", diags)
		}
		if delCalls != 2 {
			t.Fatalf("expected 2 DELETE calls (initial + one retry), got %d", delCalls)
		}
	})

	t.Run("transient wait then re-DELETE+wait OK -> success", func(t *testing.T) {
		d := newStaticIPState(t)
		var delCalls, waitCalls int
		del := delSeq(&delCalls, delResult{"act", nil})
		wait := func(ctx context.Context, activityID string) error {
			waitCalls++
			if waitCalls == 1 {
				return errVPCTransient
			}
			return nil
		}
		diags := deleteVPCStaticIPWithRetry(ctx, d, del, noList, wait, noSleep, vpcIsTransient)
		if diags.HasError() {
			t.Fatalf("a transient delete must be retried to convergence, got: %v", diags)
		}
		if delCalls != 2 || waitCalls != 2 {
			t.Fatalf("expected 2 DELETE + 2 wait, got %d/%d", delCalls, waitCalls)
		}
	})

	t.Run("persistent transient wait -> bounded, fails", func(t *testing.T) {
		d := newStaticIPState(t)
		var delCalls int
		del := delSeq(&delCalls, delResult{"act", nil})
		wait := func(ctx context.Context, activityID string) error { return errVPCTransient }
		diags := deleteVPCStaticIPWithRetry(ctx, d, del, noList, wait, noSleep, vpcIsTransient)
		if !diags.HasError() {
			t.Fatal("a persistent transient delete must eventually fail (bounded)")
		}
		if delCalls != maxTransientVPCAttempts {
			t.Fatalf("expected exactly %d DELETE calls (bounded), got %d", maxTransientVPCAttempts, delCalls)
		}
	})

	t.Run("non-transient wait failure is NOT retried", func(t *testing.T) {
		d := newStaticIPState(t)
		var delCalls int
		del := delSeq(&delCalls, delResult{"act", nil})
		wait := func(ctx context.Context, activityID string) error { return errVPCPermanent }
		diags := deleteVPCStaticIPWithRetry(ctx, d, del, noList, wait, noSleep, vpcIsTransient)
		if !diags.HasError() {
			t.Fatal("a non-transient delete failure must fail")
		}
		if delCalls != 1 {
			t.Fatalf("a non-transient failure must NOT be retried, got %d DELETE calls", delCalls)
		}
	})
}

func TestUpdateVPCStaticIPRetriesTransient502(t *testing.T) {
	ctx := context.Background()
	// newStaticIPState seeds mac=00:50:56:ab:cd:ef, resource_description="seeded";
	// those are the DESIRED values d.Get returns in the update path.
	const desiredMAC = "00:50:56:ab:cd:ef"

	live := func(desc string) *client.StaticIP {
		return &client.StaticIP{ID: "si-1", MacAddress: desiredMAC, ResourceDescription: siDescPtr(desc)}
	}
	liveMAC := func(mac, desc string) *client.StaticIP {
		return &client.StaticIP{ID: "si-1", MacAddress: mac, ResourceDescription: siDescPtr(desc)}
	}

	t.Run("transient wait then live already converged -> success, NO second PATCH", func(t *testing.T) {
		// Attempt 1 sees a diverged description -> PATCH; wait fails transient.
		// The PATCH actually applied: attempt 2 re-reads the converged live and
		// returns success WITHOUT a second PATCH.
		d := newStaticIPState(t)
		var readCalls, updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				readCalls++
				if readCalls == 1 {
					return live("old"), nil // diverged -> PATCH
				}
				return live("seeded"), nil // converged after the (transiently-reported) PATCH
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return errVPCTransient },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a PATCH that converged platform-side after a transient failure must succeed, got: %v", diags)
		}
		if updateCalls != 1 {
			t.Fatalf("a converged update must NOT be re-PATCHed on retry, got %d PATCHes; kills the re-PATCH-when-converged / rebuild-from-state mutant", updateCalls)
		}
	})

	t.Run("transient wait then still diverged -> re-PATCH, converges", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls, waitCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live("old"), nil },
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: func(ctx context.Context, activityID string) error {
				waitCalls++
				if waitCalls == 1 {
					return errVPCTransient
				}
				return nil
			},
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a still-diverged static IP must be re-PATCHed to convergence, got: %v", diags)
		}
		if updateCalls != 2 {
			t.Fatalf("expected 2 PATCHes (initial + one retry while still diverged), got %d", updateCalls)
		}
	})

	t.Run("converged from the start -> zero PATCH", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live("seeded"), nil },
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return nil },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("an already-converged update must succeed, got: %v", diags)
		}
		if updateCalls != 0 {
			t.Fatalf("a converged live must NOT be PATCHed, got %d PATCHes", updateCalls)
		}
	})

	t.Run("nil/ambiguous read fails closed (no PATCH)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return nil, nil }, // 403/absent
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return nil },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a nil/ambiguous read must FAIL CLOSED, never PATCH on ambiguous evidence")
		}
		if updateCalls != 0 {
			t.Fatalf("a fail-closed update must NOT PATCH, got %d PATCHes; kills the PATCH-on-ambiguous-read mutant", updateCalls)
		}
	})

	t.Run("id-inconsistent read fails closed (no PATCH)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return &client.StaticIP{ID: "someone-else", MacAddress: desiredMAC}, nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return nil },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("an id-inconsistent read must FAIL CLOSED, never PATCH")
		}
		if updateCalls != 0 {
			t.Fatalf("a fail-closed update must NOT PATCH, got %d PATCHes", updateCalls)
		}
	})

	t.Run("MAC format-only difference (dash + uppercase) -> zero PATCH", func(t *testing.T) {
		// desired = 00:50:56:ab:cd:ef ; live = the SAME MAC, dash-separated and
		// uppercased. normMAC canonicalises both -> no diff -> no PATCH.
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return liveMAC("00-50-56-AB-CD-EF", "seeded"), nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return nil },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a format-only MAC difference must converge cleanly, got: %v", diags)
		}
		if updateCalls != 0 {
			t.Fatalf("a format-only MAC difference must NOT PATCH (normalised equal), got %d PATCHes", updateCalls)
		}
	})

	t.Run("genuinely different MAC -> PATCH", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return liveMAC("00:50:56:00:00:01", "seeded"), nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait:        func(ctx context.Context, activityID string) error { return nil },
			retrySleep:  noSleep,
			isTransient: vpcIsTransient,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a genuinely different MAC must PATCH cleanly, got: %v", diags)
		}
		if updateCalls != 1 {
			t.Fatalf("a genuinely different MAC must PATCH exactly once, got %d PATCHes", updateCalls)
		}
	})
}
