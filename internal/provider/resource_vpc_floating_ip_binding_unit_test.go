package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	testFIPID    = "fip-1"
	testStaticID = "si-1"
	testBindID   = "fip-1:si-1"
)

// noSleep is the zero-sleep seam used by every test: the bounded confirmation
// retries never wait in unit tests.
func noSleep(ctx context.Context, attempt int) error { return nil }

// bindingState builds a ResourceData for an existing binding in the state, with
// the in/out attributes seeded.
func bindingState(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
	d.SetId(testBindID)
	for k, v := range map[string]string{
		"floating_ip_id":      testFIPID,
		"static_ip_id":        testStaticID,
		"floating_ip_address": "198.51.100.1",
		"static_ip_address":   "10.0.1.5",
	} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// emptyBindingState builds a ResourceData with no id (the create entry point).
func emptyBindingState(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
	for k, v := range map[string]string{
		"floating_ip_id": testFIPID,
		"static_ip_id":   testStaticID,
	} {
		if err := d.Set(k, v); err != nil {
			t.Fatalf("seeding %q: %v", k, err)
		}
	}
	return d
}

// boundFIP is a floating IP bound to the target static IP.
func boundFIP() *client.FloatingIP {
	return &client.FloatingIP{
		ID:        testFIPID,
		IPAddress: "198.51.100.1",
		StaticIP:  &client.FloatingIPStaticIP{ID: testStaticID, Address: "10.0.1.5"},
	}
}

// unboundFIP is a floating IP that exists but is not bound.
func unboundFIP() *client.FloatingIP {
	return &client.FloatingIP{ID: testFIPID, IPAddress: "198.51.100.1"}
}

// otherBoundFIP is a floating IP bound to a DIFFERENT static IP.
func otherBoundFIP() *client.FloatingIP {
	return &client.FloatingIP{
		ID:        testFIPID,
		IPAddress: "198.51.100.1",
		StaticIP:  &client.FloatingIPStaticIP{ID: "si-OTHER", Address: "10.0.9.9"},
	}
}

// mismatchedIDBoundFIP is a structurally inconsistent per-id read: a 200 body
// that claims to be bound to OUR static IP but whose own id is a DIFFERENT
// floating IP. Trusting it would let create/import/delete treat a wrong FIP as
// positive evidence — the #312 R7 id-match guard must reject it.
func mismatchedIDBoundFIP() *client.FloatingIP {
	return &client.FloatingIP{
		ID:        "fip-OTHER",
		IPAddress: "198.51.100.1",
		StaticIP:  &client.FloatingIPStaticIP{ID: testStaticID, Address: "10.0.1.5"},
	}
}

// emptyIDBoundFIP is a per-id read with an EMPTY id but otherwise looking like
// our pair. The id-match guard must reject an empty id as unusable evidence.
func emptyIDBoundFIP() *client.FloatingIP {
	return &client.FloatingIP{
		ID:        "",
		IPAddress: "198.51.100.1",
		StaticIP:  &client.FloatingIPStaticIP{ID: testStaticID, Address: "10.0.1.5"},
	}
}

// readSeq returns a read func that yields the supplied results in order, then
// repeats the last one. It counts the number of calls.
func readSeq(calls *int, results ...struct {
	fip *client.FloatingIP
	err error
}) func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
	return func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
		i := *calls
		*calls++
		if i >= len(results) {
			i = len(results) - 1
		}
		return results[i].fip, results[i].err
	}
}

type readResult = struct {
	fip *client.FloatingIP
	err error
}

// --- Create ---------------------------------------------------------------

// TestCreateVPCFloatingIPBinding pins the FAIL-CLOSED create contract: mutate
// only on positive 200 evidence; set the id only after the binding is confirmed
// converged; never clobber an out-of-band binding.
func TestCreateVPCFloatingIPBinding(t *testing.T) {
	ctx := context.Background()
	okWait := func(ctx context.Context, activityID string) error { return nil }
	noCorroborate := func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
		return client.FloatingIPBindingInconclusive, errors.New("corroborate must not be reached")
	}

	t.Run("pre-existing SAME pair is adopted (zero bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return boundFIP(), nil },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "a", nil },
			wait:        okWait,
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("adopting an existing same-pair binding must succeed, got: %v", diags)
		}
		if bindCalls != 0 {
			t.Fatalf("an already-bound same pair must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set on adoption, got %q", d.Id())
		}
		if d.Get("static_ip_address").(string) != "10.0.1.5" {
			t.Fatalf("computed attrs must be refreshed on adoption, got %q", d.Get("static_ip_address"))
		}
	})

	t.Run("pre-existing DIFFERENT pair is an anti-clobber error (zero bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return otherBoundFIP(), nil },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "a", nil },
			wait:        okWait,
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("binding a FIP already bound to a DIFFERENT static IP must fail (anti-clobber)")
		}
		if bindCalls != 0 {
			t.Fatalf("anti-clobber must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a failed create must not set the id, got %q", d.Id())
		}
	})

	t.Run("pre-bind 403/ambiguous + inconclusive listing is FAIL-CLOSED (zero bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403-as-absent
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "a", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an ambiguous pre-bind read with an inconclusive listing must FAIL CLOSED")
		}
		if bindCalls != 0 {
			t.Fatalf("a fail-closed pre-bind must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a fail-closed create must not set the id, got %q", d.Id())
		}
	})

	t.Run("pre-bind 403 but listing positively shows UNBOUND -> bind proceeds and converges", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			// First read (pre-bind) is ambiguous; the post-bind confirmation read
			// shows the converged pair.
			read: readSeq(&readCalls, readResult{nil, nil}, readResult{boundFIP(), nil}),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				// Genuinely UNBOUND (staticIp nil): the ONLY state that unlocks bind.
				return client.FloatingIPBindingUnbound, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a positively-unbound FIP must allow the bind to proceed, got: %v", diags)
		}
		if bindCalls != 1 {
			t.Fatalf("expected exactly one bind POST, got %d", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set after a converged bind, got %q", d.Id())
		}
	})

	t.Run("pre-bind 403 + listing shows the FIP bound to a DIFFERENT static IP -> anti-clobber error, NO id, ZERO bind POST", func(t *testing.T) {
		// F1 CRITICAL: the forbidden bind-on-bound-elsewhere path. The per-id read
		// is ambiguous (403/nil), and the strict listing positively shows the FIP
		// is bound to a DIFFERENT static IP. Create MUST fail closed, set NO id, and
		// emit ZERO bind POST — it must NOT rely on the API to reject the bind.
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403/ambiguous
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingBoundToOther, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a FIP corroborated as bound to a DIFFERENT static IP must FAIL CLOSED (anti-clobber)")
		}
		if bindCalls != 0 {
			t.Fatalf("anti-clobber via the listing must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("an anti-clobber create must not set the id, got %q", d.Id())
		}
	})

	t.Run("pre-bind 403 + listing shows our pair (BoundToTarget) -> adopt, ZERO bind POST", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			// Pre-bind read ambiguous (403/nil); then the confirmAndSet read (after
			// the BoundToTarget corroboration) converges to our pair.
			read: readSeq(&readCalls, readResult{nil, nil}, readResult{boundFIP(), nil}),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingBoundToTarget, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("adopting via the listing (BoundToTarget) must succeed, got: %v", diags)
		}
		if bindCalls != 0 {
			t.Fatalf("adoption via the listing must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set on adoption, got %q", d.Id())
		}
	})

	t.Run("unbound -> bind, then confirmation retry (negative then positive) succeeds", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls, readCalls int
		funcs := vpcFloatingIPBindingFuncs{
			// pre-bind: unbound; post-bind confirmation: negative once, then bound.
			read: readSeq(&readCalls,
				readResult{unboundFIP(), nil}, // pre-bind
				readResult{unboundFIP(), nil}, // confirm attempt 1: not yet converged
				readResult{boundFIP(), nil},   // confirm attempt 2: converged
			),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait:        okWait,
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a bind that converges on the second confirmation read must succeed, got: %v", diags)
		}
		if bindCalls != 1 {
			t.Fatalf("expected exactly one bind POST, got %d", bindCalls)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be set only after convergence, got %q", d.Id())
		}
	})

	t.Run("confirmation budget exhausted -> error, NO id set", func(t *testing.T) {
		d := emptyBindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			// pre-bind unbound; every confirmation read stays negative.
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:        okWait,
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a binding that never converges must error after the confirmation budget")
		}
		if d.Id() != "" {
			t.Fatalf("an unconfirmed binding must NOT set the id (safe recovery), got %q", d.Id())
		}
	})

	t.Run("a bind activity FAILED is an error, NO id set", func(t *testing.T) {
		d := emptyBindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:        func(ctx context.Context, activityID string) error { return errors.New("activity failed") },
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a failed bind activity must surface as an error")
		}
		if d.Id() != "" {
			t.Fatalf("a failed bind must not set the id, got %q", d.Id())
		}
	})

	t.Run("a pre-bind read error is FAIL-CLOSED (zero bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, errors.New("boom") },
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "a", nil },
			wait:        okWait,
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a hard pre-bind read error must fail closed")
		}
		if bindCalls != 0 {
			t.Fatalf("a read error must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
	})

	// F2 — the per-id pre-bind read must carry EXACTLY fipID to be usable as
	// evidence. A mismatched/empty-id 200 body must NOT be taken as adopt /
	// anti-clobber / proceed-to-bind: it is routed like an ambiguous read (to the
	// strict listing). With an Inconclusive listing the whole thing FAILS CLOSED
	// with ZERO bind POST and NO id.
	t.Run("pre-bind read with a MISMATCHED id is not usable evidence -> fail closed via listing (ZERO bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return mismatchedIDBoundFIP(), nil
			},
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a mismatched-id pre-bind read must NOT be trusted; with an inconclusive listing it must FAIL CLOSED")
		}
		if bindCalls != 0 {
			t.Fatalf("a mismatched-id read must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a fail-closed create must not set the id, got %q", d.Id())
		}
	})

	t.Run("pre-bind read with an EMPTY id is not usable evidence -> fail closed via listing (ZERO bind POST)", func(t *testing.T) {
		d := emptyBindingState(t)
		var bindCalls int
		funcs := vpcFloatingIPBindingFuncs{
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return emptyIDBoundFIP(), nil },
			bind: func(ctx context.Context, fipID, staticID string) (string, error) { bindCalls++; return "act", nil },
			wait: okWait,
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := createVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an empty-id pre-bind read must NOT be trusted; with an inconclusive listing it must FAIL CLOSED")
		}
		if bindCalls != 0 {
			t.Fatalf("an empty-id read must NOT issue a bind POST, got %d bind calls", bindCalls)
		}
		if d.Id() != "" {
			t.Fatalf("a fail-closed create must not set the id, got %q", d.Id())
		}
	})
}

// --- Read -----------------------------------------------------------------

// TestReadVPCFloatingIPBinding pins the read state-safety contract: a resource
// is dropped ONLY on a STABLE negative (the pair provably absent); an ambiguous
// (403/nil) read NEVER drops; a transient negative just before convergence does
// not drop.
func TestReadVPCFloatingIPBinding(t *testing.T) {
	ctx := context.Background()

	t.Run("pair present refreshes the computed attrs and keeps the id", func(t *testing.T) {
		d := bindingState(t)
		fresh := &client.FloatingIP{
			ID:        testFIPID,
			IPAddress: "198.51.100.42",
			StaticIP:  &client.FloatingIPStaticIP{ID: testStaticID, Address: "10.0.1.99"},
		}
		funcs := vpcFloatingIPBindingFuncs{
			read:  func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return fresh, nil },
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a present pair must refresh cleanly, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("the id must be preserved, got %q", d.Id())
		}
		if d.Get("floating_ip_address").(string) != "198.51.100.42" || d.Get("static_ip_address").(string) != "10.0.1.99" {
			t.Fatalf("computed attrs must be refreshed, got fip=%q static=%q", d.Get("floating_ip_address"), d.Get("static_ip_address"))
		}
	})

	t.Run("transient negative then present keeps the binding (no drop)", func(t *testing.T) {
		d := bindingState(t)
		var calls int
		funcs := vpcFloatingIPBindingFuncs{
			// First read: present but unbound (stale-200 just after bind). Second
			// read: converged.
			read:  readSeq(&calls, readResult{unboundFIP(), nil}, readResult{boundFIP(), nil}),
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a transient negative that resolves must not error, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("a transient negative must NOT drop the binding, got id %q", d.Id())
		}
	})

	t.Run("stable negative after the bounded retry DROPS the binding", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			// Always present and unbound: a stable negative -> positive absence of
			// THIS pair -> drop.
			read:  func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a stable-negative read must drop cleanly, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("a stable negative must drop the binding (SetId(\"\")), got id %q", d.Id())
		}
	})

	t.Run("a 403/nil read KEEPS the binding (never drops on ambiguity)", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			read:  func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403-as-absent
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a 403/nil read must keep the state without error, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("a 403/nil read must NEVER drop the binding, got id %q", d.Id())
		}
	})

	t.Run("a read becoming ambiguous mid-confirmation KEEPS the binding (no drop)", func(t *testing.T) {
		d := bindingState(t)
		var calls int
		funcs := vpcFloatingIPBindingFuncs{
			// present-unbound first, then 403/nil -> ambiguous -> must not drop.
			read:  readSeq(&calls, readResult{unboundFIP(), nil}, readResult{nil, nil}),
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("an ambiguity mid-confirmation must keep the state without error, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("an ambiguity mid-confirmation must NOT drop, got id %q", d.Id())
		}
	})

	t.Run("a hard read error fails closed and keeps the id", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			read:  func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, errors.New("boom") },
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a hard read error must surface as a diagnostic")
		}
		if d.Id() != testBindID {
			t.Fatalf("a hard read error must keep the id, got %q", d.Id())
		}
	})

	// F2 — a per-id read with a MISMATCHED id (a 200 body for a different FIP that
	// happens to look unbound, or even claims our pair) must NOT be trusted as
	// evidence. In particular it must NEVER trigger SetId("") — it is treated like
	// an ambiguous read and the binding is kept.
	t.Run("a MISMATCHED-id read NEVER drops the binding (treated as ambiguous)", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			// Body for a DIFFERENT FIP, unbound: a naive read would treat this as a
			// stable negative for OUR FIP and drop. The guard must keep state.
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return &client.FloatingIP{ID: "fip-OTHER"}, nil
			},
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a mismatched-id read must keep the state without error, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("a mismatched-id read must NEVER drop the binding, got id %q", d.Id())
		}
	})

	t.Run("an EMPTY-id read NEVER drops the binding (treated as ambiguous)", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			read:  func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return emptyIDBoundFIP(), nil },
			sleep: noSleep,
		}
		diags := readVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("an empty-id read must keep the state without error, got: %v", diags)
		}
		if d.Id() != testBindID {
			t.Fatalf("an empty-id read must NEVER drop the binding, got id %q", d.Id())
		}
	})
}

// --- Delete ---------------------------------------------------------------

// TestDeleteVPCFloatingIPBinding pins the unbind state-safety contract: removal
// only on STRICT positive confirmation that the pair is gone. A 403 alone is
// NEVER "gone".
func TestDeleteVPCFloatingIPBinding(t *testing.T) {
	ctx := context.Background()
	okWait := func(ctx context.Context, activityID string) error { return nil }
	noCorroborate := func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
		return client.FloatingIPBindingInconclusive, errors.New("corroborate must not be reached")
	}

	t.Run("unbind then read-200-not-our-pair is a success", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:        okWait,
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("an unbind confirmed by a 200-not-our-pair read must succeed, got: %v", diags)
		}
	})

	t.Run("unbind then read-403 + strict-List UNBOUND is a success", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read:   func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403/ambiguous
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingUnbound, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a 403 read corroborated by an UNBOUND listing must succeed, got: %v", diags)
		}
	})

	t.Run("unbind then read-403 + strict-List BoundToOther is a success (no longer OUR pair)", func(t *testing.T) {
		// The FIP was re-bound elsewhere (or the listing shows a different static
		// IP): it is no longer bound to OUR pair, so OUR unbind took effect.
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read:   func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403/ambiguous
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingBoundToOther, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a 403 read corroborated by a BoundToOther listing must succeed (no longer our pair), got: %v", diags)
		}
	})

	t.Run("unbind then read-403 + strict-List BoundToTarget FAILS CLOSED (still our pair)", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read:   func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // 403/ambiguous
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingBoundToTarget, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a 403 read corroborated as STILL bound to our pair must FAIL CLOSED (keep state)")
		}
	})

	t.Run("unbind then read-403 + List INCONCLUSIVE FAILS CLOSED", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read:   func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil },
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a 403 read with an inconclusive listing must FAIL CLOSED (no removal)")
		}
	})

	t.Run("unbind then read-403 + List FAILED FAILS CLOSED", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read:   func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil },
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, errors.New("listing 500")
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a 403 read with a failing listing must FAIL CLOSED")
		}
	})

	t.Run("unbind then still-bound read FAILS CLOSED (the unbind did not take)", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:        okWait,
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return boundFIP(), nil },
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an unbind whose read still shows our pair must FAIL CLOSED")
		}
	})

	// F2 — a per-id read with a MISMATCHED id must NOT be accepted as "gone". A
	// naive path would see a different-FIP body that is "not our pair" and treat
	// the unbind as a success. The guard routes it to the strict listing; with an
	// Inconclusive listing the delete FAILS CLOSED (keeps the binding).
	t.Run("unbind then MISMATCHED-id read is NOT accepted as gone -> fail closed via listing", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			// A 200 for a DIFFERENT FIP, unbound: must NOT be read as "our pair is gone".
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return &client.FloatingIP{ID: "fip-OTHER"}, nil
			},
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a mismatched-id read must NOT be accepted as gone; with an inconclusive listing it must FAIL CLOSED")
		}
		if d.Id() != testBindID {
			t.Fatalf("a fail-closed delete must keep the binding id, got %q", d.Id())
		}
	})

	t.Run("unbind then EMPTY-id read is NOT accepted as gone -> fail closed via listing", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   okWait,
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return &client.FloatingIP{ID: ""}, nil
			},
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("an empty-id read must NOT be accepted as gone; with an inconclusive listing it must FAIL CLOSED")
		}
		if d.Id() != testBindID {
			t.Fatalf("a fail-closed delete must keep the binding id, got %q", d.Id())
		}
	})

	t.Run("a 404 on the unbind CALL is idempotent ONLY after positive confirmation", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
				return "", client.StatusError{Code: http.StatusNotFound}
			},
			wait:        func(ctx context.Context, activityID string) error { return errors.New("wait must not be reached") },
			read:        func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return unboundFIP(), nil },
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if diags.HasError() {
			t.Fatalf("a 404 unbind confirmed by a 200-not-our-pair read must succeed, got: %v", diags)
		}
	})

	t.Run("a 403 on the unbind CALL without positive confirmation FAILS CLOSED", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
				return "", client.StatusError{Code: http.StatusForbidden}
			},
			wait: func(ctx context.Context, activityID string) error { return errors.New("wait must not be reached") },
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) { return nil, nil }, // ambiguous
			corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
				return client.FloatingIPBindingInconclusive, nil
			},
			sleep: noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a 403 unbind call without a strict positive confirmation must FAIL CLOSED")
		}
	})

	t.Run("a 500 on the unbind CALL is an error", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
				return "", client.StatusError{Code: http.StatusInternalServerError}
			},
			wait: func(ctx context.Context, activityID string) error { return errors.New("wait must not be reached") },
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return nil, errors.New("read must not be reached")
			},
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a 500 on the unbind call must surface as an error")
		}
	})

	t.Run("a failed unbind activity is an error", func(t *testing.T) {
		d := bindingState(t)
		funcs := vpcFloatingIPBindingFuncs{
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) { return "act", nil },
			wait:   func(ctx context.Context, activityID string) error { return errors.New("activity failed") },
			read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
				return nil, errors.New("read must not be reached")
			},
			corroborate: noCorroborate,
			sleep:       noSleep,
		}
		diags := deleteVPCFloatingIPBinding(ctx, d, testFIPID, testStaticID, funcs)
		if !diags.HasError() {
			t.Fatal("a failed unbind activity must surface as an error")
		}
	})
}

// --- ID parsing & import --------------------------------------------------

// TestSplitFloatingIPBindingID pins the composite-id parser: exactly two
// non-empty halves, else an error.
func TestSplitFloatingIPBindingID(t *testing.T) {
	good := []struct{ id, fip, static string }{
		{"fip-1:si-1", "fip-1", "si-1"},
		{"a:b", "a", "b"},
	}
	for _, g := range good {
		fip, static, err := splitFloatingIPBindingID(g.id)
		if err != nil || fip != g.fip || static != g.static {
			t.Fatalf("split(%q) = (%q,%q,%v), want (%q,%q,nil)", g.id, fip, static, err, g.fip, g.static)
		}
	}
	bad := []string{"", "fip-only", ":si-1", "fip-1:", ":", "a:b:c", "fip-1:si-1:"}
	for _, b := range bad {
		if _, _, err := splitFloatingIPBindingID(b); err == nil {
			t.Fatalf("split(%q) must error (malformed composite id)", b)
		}
	}
}

func TestMakeFloatingIPBindingIDRoundTrip(t *testing.T) {
	id := makeFloatingIPBindingID("fip-9", "si-9")
	fip, static, err := splitFloatingIPBindingID(id)
	if err != nil || fip != "fip-9" || static != "si-9" {
		t.Fatalf("round-trip failed: id=%q -> (%q,%q,%v)", id, fip, static, err)
	}
}
