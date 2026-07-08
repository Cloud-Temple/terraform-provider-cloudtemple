package provider

import (
	"context"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// siDescPtr returns a *string for seeding the nullable ResourceDescription on a
// client.StaticIP fixture. Shared across the static IP update/create tests.
func siDescPtr(s string) *string { return &s }

// TestUpdateVPCStaticIPWith pins the PATCH logic. The contract:
//
//   - the PATCH body is rebuilt from a FRESH LIVE read, never the state alone, and
//     is DIFF-DRIVEN (only the genuinely-changed fields are sent);
//   - a nil/ambiguous read, an id-inconsistent read, or a hard read error all FAIL
//     CLOSED — never PATCH on ambiguous evidence;
//   - MAC comparison is canonicalised, so a pure formatting difference is NOT a diff;
//   - there is NO retry (the transient-502 retry was retired): a single failed
//     PATCH or a single failed wait surfaces an actionable diagnostic — the operator
//     re-applies (Terraform's natural retry). The "wait fails" / "PATCH fails" cases
//     pin that single-pass fail-closed behaviour.
//
// newStaticIPState seeds the DESIRED config the update path reads from d.Get:
// mac=00:50:56:ab:cd:ef, resource_description="seeded".
func TestUpdateVPCStaticIPWith(t *testing.T) {
	ctx := context.Background()
	const desiredMAC = "00:50:56:ab:cd:ef"
	const desiredDesc = "seeded"

	// live builds a live read whose MAC already equals the desired MAC; only the
	// description varies. liveMAC varies the MAC too. Both carry source="custom":
	// the write-path source guard (#311) admits a PATCH only on a positively-custom
	// live read, so the happy-path fixtures must be custom.
	live := func(desc string) *client.StaticIP {
		return &client.StaticIP{ID: "si-1", MacAddress: desiredMAC, ResourceDescription: siDescPtr(desc), Source: "custom"}
	}
	liveMAC := func(mac, desc string) *client.StaticIP {
		return &client.StaticIP{ID: "si-1", MacAddress: mac, ResourceDescription: siDescPtr(desc), Source: "custom"}
	}
	okWait := func(ctx context.Context, activityID string) error { return nil }

	t.Run("converged from the start -> zero PATCH", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live(desiredDesc), nil },
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("an already-converged update must succeed, got: %v", diags)
		}
		if updateCalls != 0 {
			t.Fatalf("a converged live must NOT be PATCHed, got %d PATCHes", updateCalls)
		}
	})

	t.Run("diverged description -> exactly one PATCH carrying only the description", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		var gotReq *client.UpdateStaticIPRequest
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live("old"), nil }, // desc diverges
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				gotReq = req
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a diverged description must PATCH cleanly, got: %v", diags)
		}
		if updateCalls != 1 {
			t.Fatalf("a diverged description must PATCH exactly once, got %d", updateCalls)
		}
		// Diff-driven body: ONLY the description changed, so the MAC must be absent.
		if gotReq.ResourceDescription == nil || *gotReq.ResourceDescription != desiredDesc {
			t.Fatalf("the PATCH must carry resourceDescription=%q, got %v", desiredDesc, gotReq.ResourceDescription)
		}
		if gotReq.MacAddress != nil {
			t.Fatalf("the PATCH must NOT carry the (unchanged) MAC, got %v; kills the rebuild-from-state mutant", *gotReq.MacAddress)
		}
	})

	t.Run("genuinely different MAC -> exactly one PATCH carrying only the MAC", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		var gotReq *client.UpdateStaticIPRequest
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return liveMAC("00:50:56:00:00:01", desiredDesc), nil // MAC diverges, desc matches
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				gotReq = req
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a genuinely different MAC must PATCH cleanly, got: %v", diags)
		}
		if updateCalls != 1 {
			t.Fatalf("a genuinely different MAC must PATCH exactly once, got %d", updateCalls)
		}
		if gotReq.MacAddress == nil || *gotReq.MacAddress != desiredMAC {
			t.Fatalf("the PATCH must carry macAddress=%q, got %v", desiredMAC, gotReq.MacAddress)
		}
		if gotReq.ResourceDescription != nil {
			t.Fatalf("the PATCH must NOT carry the (unchanged) description, got %v", *gotReq.ResourceDescription)
		}
	})

	t.Run("MAC format-only difference (dash + uppercase) -> zero PATCH", func(t *testing.T) {
		// desired = 00:50:56:ab:cd:ef ; live = the SAME MAC, dash-separated and
		// uppercased. normMAC canonicalises both -> no diff -> no PATCH.
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return liveMAC("00-50-56-AB-CD-EF", desiredDesc), nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if diags.HasError() {
			t.Fatalf("a format-only MAC difference must converge cleanly, got: %v", diags)
		}
		if updateCalls != 0 {
			t.Fatalf("a format-only MAC difference must NOT PATCH (normalised equal), got %d PATCHes", updateCalls)
		}
	})

	t.Run("nil/ambiguous read fails closed (no PATCH)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return nil, nil }, // 404 absent (post-#384 a 403 would error)
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
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
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("an id-inconsistent read must FAIL CLOSED, never PATCH")
		}
		if updateCalls != 0 {
			t.Fatalf("a fail-closed update must NOT PATCH, got %d PATCHes", updateCalls)
		}
	})

	// #311 write-path guard: a non-custom live read must FAIL CLOSED before any
	// PATCH. Both fixtures DIVERGE the description, so WITHOUT the guard the update
	// would issue a PATCH (and succeed on okWait) — the assertions on HasError AND
	// zero PATCH/wait are therefore non-complacent: they red if the guard is removed.
	t.Run("a non-custom (xoa) live read fails closed before any PATCH", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls, waitCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return &client.StaticIP{ID: "si-1", MacAddress: desiredMAC, ResourceDescription: siDescPtr("old"), Source: "xoa"}, nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: func(ctx context.Context, activityID string) error { waitCalls++; return nil },
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a non-custom (xoa) live read must FAIL CLOSED, never PATCH a static IP Terraform cannot delete")
		}
		if updateCalls != 0 || waitCalls != 0 {
			t.Fatalf("the source guard must prevent any PATCH/wait, got %d PATCH / %d wait", updateCalls, waitCalls)
		}
	})

	t.Run("an unproven empty-source live read fails closed before any PATCH", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				return &client.StaticIP{ID: "si-1", MacAddress: desiredMAC, ResourceDescription: siDescPtr("old"), Source: ""}, nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("an empty source is not proof of a custom static IP; update must FAIL CLOSED, never PATCH on unproven ownership")
		}
		if updateCalls != 0 {
			t.Fatalf("an unproven-source update must NOT PATCH, got %d PATCHes", updateCalls)
		}
	})

	// Companion that pins the guard PRECEDES the diff: a non-custom live read whose
	// fields ALREADY MATCH the desired config (converged) must STILL fail closed.
	// Without the guard this hits the no-PATCH converged path and returns success;
	// with the guard misplaced AFTER the diff it would also return success — so this
	// reds on BOTH the guard's removal AND its mispositioning.
	t.Run("a converged-but-non-custom live read still fails closed (guard precedes the diff)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) {
				// MAC + description both EQUAL the desired config -> converged (no diff),
				// but the source is non-custom.
				return &client.StaticIP{ID: "si-1", MacAddress: desiredMAC, ResourceDescription: siDescPtr(desiredDesc), Source: "xoa"}, nil
			},
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a non-custom live read must FAIL CLOSED even when converged; the source guard must precede the diff/no-PATCH path")
		}
		if updateCalls != 0 {
			t.Fatalf("a fail-closed update must NOT PATCH, got %d PATCHes", updateCalls)
		}
	})

	t.Run("a hard read error fails closed (no PATCH)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: siRead(nil, context.DeadlineExceeded),
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: okWait,
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a hard read error must FAIL CLOSED before any PATCH")
		}
		if updateCalls != 0 {
			t.Fatalf("a fail-closed update must NOT PATCH on a read error, got %d PATCHes", updateCalls)
		}
	})

	t.Run("a PATCH error fails (wait not reached)", func(t *testing.T) {
		d := newStaticIPState(t)
		var waitCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live("old"), nil }, // diverged -> PATCH
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				return "", context.DeadlineExceeded
			},
			wait: func(ctx context.Context, activityID string) error { waitCalls++; return nil },
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a failed PATCH must surface as an error")
		}
		if waitCalls != 0 {
			t.Fatalf("a failed PATCH must not wait on a non-existent activity, got %d wait calls", waitCalls)
		}
	})

	// THE no-retry behaviour: a single failed update activity fails closed in one
	// pass. The retired retry would have re-PATCHed here; without it, the failure
	// surfaces an actionable diagnostic and the operator re-applies. Non-complacent:
	// a mutant that swallowed the wait error (reported success) reds here.
	t.Run("a failed update activity fails closed in a single pass (no retry)", func(t *testing.T) {
		d := newStaticIPState(t)
		var updateCalls, waitCalls int
		funcs := vpcStaticIPUpdateFuncs{
			read: func(ctx context.Context, id string) (*client.StaticIP, error) { return live("old"), nil }, // diverged -> PATCH
			update: func(ctx context.Context, id string, req *client.UpdateStaticIPRequest) (string, error) {
				updateCalls++
				return "act", nil
			},
			wait: func(ctx context.Context, activityID string) error { waitCalls++; return context.DeadlineExceeded },
		}
		diags := updateVPCStaticIPWith(ctx, d, funcs)
		if !diags.HasError() {
			t.Fatal("a failed update activity must surface as an error, not a silent success")
		}
		if updateCalls != 1 || waitCalls != 1 {
			t.Fatalf("the update path must be single-pass (no retry): expected 1 PATCH + 1 wait, got %d/%d", updateCalls, waitCalls)
		}
	})
}
