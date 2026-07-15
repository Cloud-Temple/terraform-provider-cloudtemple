package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ---- seam fakes -----------------------------------------------------------

type bindResolveResult struct {
	state client.FloatingIPBindingState
	fip   *client.FloatingIP
	found bool
	err   error
}

func bindResolveOnce(r bindResolveResult) bindingResolveFunc {
	return func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, *client.FloatingIP, bool, error) {
		return r.state, r.fip, r.found, r.err
	}
}

// bindResolveSeq yields a different programmed result on each call — to drive the
// pre-read/confirm (create) and preflight/final (delete) reads independently.
func bindResolveSeq(t *testing.T, results ...bindResolveResult) bindingResolveFunc {
	t.Helper()
	calls := 0
	return func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, *client.FloatingIP, bool, error) {
		if calls >= len(results) {
			t.Fatalf("resolve called %d time(s), only %d programmed", calls+1, len(results))
		}
		r := results[calls]
		calls++
		return r.state, r.fip, r.found, r.err
	}
}

func bindFatal(t *testing.T) func(context.Context, string, string) (string, error) {
	return func(ctx context.Context, fipID, staticID string) (string, error) {
		t.Fatal("Bind must not be called on this path")
		return "", nil
	}
}

func unbindFatal(t *testing.T) func(context.Context, string, string) (string, error) {
	return func(ctx context.Context, fipID, staticID string) (string, error) {
		t.Fatal("Unbind must not be called on this path")
		return "", nil
	}
}

func bindWaitFatal(t *testing.T) vpcActivityWaitFunc {
	return func(ctx context.Context, activityID string) error {
		t.Fatal("wait must not be called on this path (empty activity id / no async)")
		return nil
	}
}

func bindWaitOK() vpcActivityWaitFunc {
	return func(ctx context.Context, activityID string) error { return nil }
}
func transientYes(error) bool { return true }
func transientNo(error) bool  { return false }

func newBindingData(t *testing.T, withID bool) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
	if err := d.Set("floating_ip_id", "fip-1"); err != nil {
		t.Fatalf("seeding floating_ip_id: %v", err)
	}
	if err := d.Set("static_ip_id", "si-1"); err != nil {
		t.Fatalf("seeding static_ip_id: %v", err)
	}
	if withID {
		d.SetId(bindingID("fip-1", "si-1"))
	}
	return d
}

func boundToTarget(addr string) bindResolveResult {
	return bindResolveResult{state: client.FloatingIPBindingBoundToTarget, fip: &client.FloatingIP{ID: "fip-1", IPAddress: addr}, found: true}
}

// ---- create ---------------------------------------------------------------

func TestCreateVPCFloatingIPBindingWith(t *testing.T) {
	ctx := context.Background()

	t.Run("Unbound pre-read -> bind -> confirm BoundToTarget succeeds and flattens the address", func(t *testing.T) {
		d := newBindingData(t, false)
		var bound bool
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t,
				bindResolveResult{state: client.FloatingIPBindingUnbound, found: true},
				boundToTarget("198.51.100.7"),
			),
			bind: func(ctx context.Context, fipID, staticID string) (string, error) {
				bound = true
				return "act-bind", nil
			},
			unbind:      unbindFatal(t),
			wait:        bindWaitOK(),
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("a clean bind must succeed, got: %v", diags)
		}
		if !bound {
			t.Fatal("Bind must be called when the FIP is provably Unbound")
		}
		if d.Id() != bindingID("fip-1", "si-1") {
			t.Fatalf("the composite id must be set, got %q", d.Id())
		}
		if got := d.Get("floating_ip_address").(string); got != "198.51.100.7" {
			t.Fatalf("floating_ip_address must be flattened from the confirm read, got %q", got)
		}
	})

	// Anti-clobber: a FIP bound elsewhere, an inconclusive read, or an absent FIP must
	// NEVER be bound. A mutant that bound on BoundToOther/Inconclusive reds (bindFatal).
	for _, tc := range []struct {
		name string
		pre  bindResolveResult
	}{
		{"BoundToOther fails closed (anti-clobber)", bindResolveResult{state: client.FloatingIPBindingBoundToOther, found: true}},
		{"Inconclusive fails closed", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true}},
		{"absent FIP (404) fails closed", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: false}},
	} {
		t.Run("create: "+tc.name, func(t *testing.T) {
			d := newBindingData(t, false)
			diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
				resolve:     bindResolveOnce(tc.pre),
				bind:        bindFatal(t),
				unbind:      unbindFatal(t),
				wait:        bindWaitFatal(t),
				isTransient: transientNo,
			})
			if !diags.HasError() {
				t.Fatal("must fail closed (no bind) when the FIP is not provably free")
			}
			if d.Id() != "" {
				t.Fatalf("must NOT set an id when binding is refused, got %q", d.Id())
			}
		})
	}

	t.Run("BoundToTarget pre-read is idempotent: no bind, id set, success", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, boundToTarget("198.51.100.7"), boundToTarget("198.51.100.7")),
			bind:        bindFatal(t), // already our pair -> must NOT re-bind
			unbind:      unbindFatal(t),
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("an already-bound pair must be idempotent success, got: %v", diags)
		}
		if d.Id() != bindingID("fip-1", "si-1") {
			t.Fatalf("idempotent adopt must set the id, got %q", d.Id())
		}
	})

	t.Run("empty activity id (409 idempotent) skips wait and confirms", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}, boundToTarget("198.51.100.7")),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "", nil }, // 409 -> empty activity id
			unbind:      unbindFatal(t),
			wait:        bindWaitFatal(t), // must NOT wait on an empty activity id
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("a 409 idempotent bind must succeed without waiting, got: %v", diags)
		}
		if d.Id() == "" {
			t.Fatal("the id must be set after a confirmed (409) bind")
		}
	})

	t.Run("a bind error fails and never sets the id", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveOnce(bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "", errors.New("boom") },
			unbind:      unbindFatal(t),
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("a bind error must surface")
		}
		if d.Id() != "" {
			t.Fatalf("a bind error must NEVER set an id, got %q", d.Id())
		}
	})

	t.Run("transient wait failure + confirm BoundToTarget = landed = success", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t,
				bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}, // pre-read
				boundToTarget("198.51.100.7"),                                          // transient confirm: landed
				boundToTarget("198.51.100.7"),                                          // final confirm
			),
			bind:   func(ctx context.Context, fipID, staticID string) (string, error) { return "act-bind", nil },
			unbind: unbindFatal(t),
			wait: func(ctx context.Context, activityID string) error {
				return errors.New("None of the workers were able to respond")
			},
			isTransient: transientYes,
		})
		if diags.HasError() {
			t.Fatalf("a transient failure that actually landed (confirm BoundToTarget) must succeed, got: %v", diags)
		}
		if d.Id() == "" {
			t.Fatal("a landed-despite-transient bind must set the id")
		}
	})

	t.Run("transient wait failure + confirm NOT landed fails closed, no id", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t,
				bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}, // pre-read
				bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}, // transient confirm: NOT landed
			),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "act-bind", nil },
			unbind:      unbindFatal(t),
			wait:        func(ctx context.Context, activityID string) error { return errors.New("transient") },
			isTransient: transientYes,
		})
		if !diags.HasError() {
			t.Fatal("a transient failure that did NOT land must fail closed")
		}
		if d.Id() != "" {
			t.Fatalf("an unconfirmed bind must NOT set an id, got %q", d.Id())
		}
	})

	t.Run("a non-transient wait failure fails and never sets the id", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveOnce(bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "act-bind", nil },
			unbind:      unbindFatal(t),
			wait:        func(ctx context.Context, activityID string) error { return errors.New("permanent") },
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("a non-transient wait failure must surface")
		}
		if d.Id() != "" {
			t.Fatalf("a failed bind must NOT set an id, got %q", d.Id())
		}
	})

	// Post-bind confirm read that is Inconclusive (eventual consistency) keeps the id
	// (never orphan a just-written binding) and errors (never report success-pending).
	t.Run("confirm Inconclusive keeps the id and errors", func(t *testing.T) {
		d := newBindingData(t, false)
		diags := createVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t,
				bindResolveResult{state: client.FloatingIPBindingUnbound, found: true},
				bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true},
			),
			bind:        func(ctx context.Context, fipID, staticID string) (string, error) { return "act-bind", nil },
			unbind:      unbindFatal(t),
			wait:        bindWaitOK(),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("an inconclusive confirm must error (never report success-pending)")
		}
		if d.Id() == "" {
			t.Fatal("the just-written id must be KEPT on an inconclusive confirm (never orphan)")
		}
	})
}

// ---- read -----------------------------------------------------------------

func TestReadVPCFloatingIPBindingInto(t *testing.T) {
	ctx := context.Background()

	t.Run("refresh: a resolve error fails closed and keeps the id", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := readVPCFloatingIPBindingInto(ctx, d, bindResolveOnce(bindResolveResult{state: client.FloatingIPBindingInconclusive, err: errors.New("403")}), bindingReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a read error must fail closed")
		}
		if d.Id() == "" {
			t.Fatal("a read error must keep the id")
		}
	})

	for _, tc := range []struct {
		name     string
		res      bindResolveResult
		wantDrop bool
	}{
		{"BoundToTarget keeps", boundToTarget("198.51.100.7"), false},
		{"Unbound drops", bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}, true},
		{"BoundToOther drops", bindResolveResult{state: client.FloatingIPBindingBoundToOther, found: true}, true},
		{"authoritative 404 drops", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: false}, true},
	} {
		t.Run("refresh: "+tc.name, func(t *testing.T) {
			d := newBindingData(t, true)
			diags := readVPCFloatingIPBindingInto(ctx, d, bindResolveOnce(tc.res), bindingReadForRefresh)
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}
			dropped := d.Id() == ""
			if dropped != tc.wantDrop {
				t.Fatalf("dropped = %v, want %v (id=%q)", dropped, tc.wantDrop, d.Id())
			}
		})
	}

	t.Run("refresh: Inconclusive fails closed (keeps the id, never drops)", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := readVPCFloatingIPBindingInto(ctx, d, bindResolveOnce(bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true}), bindingReadForRefresh)
		if !diags.HasError() {
			t.Fatal("an inconclusive refresh must fail closed, never drop")
		}
		if d.Id() == "" {
			t.Fatal("an inconclusive refresh must NOT drop the resource")
		}
	})

	t.Run("readAfterWrite: BoundToTarget flattens and succeeds", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := readVPCFloatingIPBindingInto(ctx, d, bindResolveOnce(boundToTarget("203.0.113.9")), bindingReadAfterWrite)
		if diags.HasError() {
			t.Fatalf("unexpected error: %v", diags)
		}
		if got := d.Get("floating_ip_address").(string); got != "203.0.113.9" {
			t.Fatalf("floating_ip_address = %q, want 203.0.113.9", got)
		}
	})

	// readAfterWrite must NEVER drop and never report success on a non-BoundToTarget
	// state (a mutant reusing readForRefresh here would drop a just-written id).
	for _, tc := range []struct {
		name string
		res  bindResolveResult
	}{
		{"Unbound", bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}},
		{"BoundToOther", bindResolveResult{state: client.FloatingIPBindingBoundToOther, found: true}},
		{"404", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: false}},
		{"Inconclusive", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true}},
	} {
		t.Run("readAfterWrite "+tc.name+" keeps id and errors", func(t *testing.T) {
			d := newBindingData(t, true)
			diags := readVPCFloatingIPBindingInto(ctx, d, bindResolveOnce(tc.res), bindingReadAfterWrite)
			if !diags.HasError() {
				t.Fatalf("readAfterWrite must error on %s (unconfirmed bind)", tc.name)
			}
			if d.Id() == "" {
				t.Fatalf("readAfterWrite must KEEP the just-written id on %s (never orphan/drop)", tc.name)
			}
		})
	}
}

// ---- delete ---------------------------------------------------------------

func TestDeleteVPCFloatingIPBindingWith(t *testing.T) {
	ctx := context.Background()

	t.Run("BoundToTarget -> unbind -> confirm Unbound succeeds", func(t *testing.T) {
		d := newBindingData(t, true)
		var unbound bool
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t, boundToTarget(""), bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}),
			bind:    bindFatal(t),
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
				unbound = true
				return "act-unbind", nil
			},
			wait:        bindWaitOK(),
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("a confirmed unbind must succeed, got: %v", diags)
		}
		if !unbound {
			t.Fatal("Unbind must be called when the pair is BoundToTarget")
		}
	})

	// Idempotent: already not our pair (or FIP gone) -> success, NO unbind issued.
	for _, tc := range []struct {
		name string
		pre  bindResolveResult
	}{
		{"already Unbound", bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}},
		{"bound elsewhere (BoundToOther)", bindResolveResult{state: client.FloatingIPBindingBoundToOther, found: true}},
		{"FIP absent (404)", bindResolveResult{state: client.FloatingIPBindingInconclusive, found: false}},
	} {
		t.Run("delete idempotent: "+tc.name, func(t *testing.T) {
			d := newBindingData(t, true)
			diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
				resolve:     bindResolveOnce(tc.pre),
				bind:        bindFatal(t),
				unbind:      unbindFatal(t), // must NOT unbind a pair that is already not ours
				wait:        bindWaitFatal(t),
				isTransient: transientNo,
			})
			if diags.HasError() {
				t.Fatalf("idempotent delete must succeed, got: %v", diags)
			}
		})
	}

	t.Run("preflight Inconclusive fails closed, no unbind", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveOnce(bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true}),
			bind:        bindFatal(t),
			unbind:      unbindFatal(t),
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("an inconclusive preflight must fail closed (never unbind on inconclusive evidence)")
		}
	})

	t.Run("unbind error + confirm gone = idempotent success", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve: bindResolveSeq(t, boundToTarget(""), bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}),
			bind:    bindFatal(t),
			unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
				return "", errors.New("404 surfaced")
			},
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("an unbind error confirmed gone by read must succeed, got: %v", diags)
		}
	})

	t.Run("unbind error + still bound = fail closed", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, boundToTarget(""), boundToTarget("")),
			bind:        bindFatal(t),
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "", errors.New("403") },
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("an unbind error that is NOT confirmed gone must fail closed")
		}
	})

	t.Run("empty activity id (409) skips wait, confirm Unbound succeeds", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, boundToTarget(""), bindResolveResult{state: client.FloatingIPBindingUnbound, found: true}),
			bind:        bindFatal(t),
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "", nil }, // 409
			wait:        bindWaitFatal(t),
			isTransient: transientNo,
		})
		if diags.HasError() {
			t.Fatalf("a 409 unbind must succeed without waiting, got: %v", diags)
		}
	})

	t.Run("final confirm still BoundToTarget = error (unbind did not take)", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, boundToTarget(""), boundToTarget("")),
			bind:        bindFatal(t),
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "act-unbind", nil },
			wait:        bindWaitOK(),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("a final confirm still BoundToTarget means the unbind did not take -> error")
		}
	})

	t.Run("final confirm Inconclusive = fail closed", func(t *testing.T) {
		d := newBindingData(t, true)
		diags := deleteVPCFloatingIPBindingWith(ctx, d, vpcFloatingIPBindingFuncs{
			resolve:     bindResolveSeq(t, boundToTarget(""), bindResolveResult{state: client.FloatingIPBindingInconclusive, found: true}),
			bind:        bindFatal(t),
			unbind:      func(ctx context.Context, fipID, staticID string) (string, error) { return "act-unbind", nil },
			wait:        bindWaitOK(),
			isTransient: transientNo,
		})
		if !diags.HasError() {
			t.Fatal("an inconclusive final confirm must fail closed (never report a delete on inconclusive evidence)")
		}
	})
}

// ---- schema / import pins -------------------------------------------------

func TestVPCFloatingIPBindingResourceSchema(t *testing.T) {
	r := resourceVPCFloatingIPBinding()
	if r.UpdateContext != nil {
		t.Fatal("the binding resource must have NO Update (both ids are ForceNew)")
	}
	for _, k := range []string{"floating_ip_id", "static_ip_id"} {
		s, ok := r.Schema[k]
		if !ok {
			t.Fatalf("%q missing from schema", k)
		}
		if !s.Required || !s.ForceNew {
			t.Fatalf("%q must be Required + ForceNew", k)
		}
		if s.ValidateFunc == nil {
			t.Fatalf("%q must carry an IsUUID ValidateFunc", k)
		}
		if _, errs := s.ValidateFunc("not-a-uuid", k); len(errs) == 0 {
			t.Fatalf("%q must reject a non-UUID", k)
		}
	}
	addr, ok := r.Schema["floating_ip_address"]
	if !ok || !addr.Computed || addr.Required || addr.Optional {
		t.Fatal("floating_ip_address must be Computed (read-only)")
	}
}

func TestVPCFloatingIPBindingImport(t *testing.T) {
	ctx := context.Background()

	t.Run("valid composite id sets both ids", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
		d.SetId("fip-9/si-9")
		out, err := resourceVPCFloatingIPBindingImport(ctx, d, nil)
		if err != nil {
			t.Fatalf("valid import must succeed: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("import must return one ResourceData, got %d", len(out))
		}
		if d.Get("floating_ip_id").(string) != "fip-9" || d.Get("static_ip_id").(string) != "si-9" {
			t.Fatalf("import must set both ids, got %q / %q", d.Get("floating_ip_id"), d.Get("static_ip_id"))
		}
	})

	for _, bad := range []string{"fip-only", "/si", "fip/", "", "a/b/c"} {
		t.Run("invalid import id "+bad, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, resourceVPCFloatingIPBinding().Schema, map[string]interface{}{})
			d.SetId(bad)
			if _, err := resourceVPCFloatingIPBindingImport(ctx, d, nil); err == nil {
				t.Fatalf("import id %q must be rejected", bad)
			}
		})
	}
}
