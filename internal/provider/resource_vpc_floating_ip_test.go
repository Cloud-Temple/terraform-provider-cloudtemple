package provider

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ---- seam fakes -----------------------------------------------------------

func fipResolve(fip *client.FloatingIP, found bool, err error) vpcFloatingIPResolveFunc {
	return func(ctx context.Context, id string) (*client.FloatingIP, bool, error) {
		return fip, found, err
	}
}

type fipResolveResult struct {
	fip   *client.FloatingIP
	found bool
	err   error
}

// fipResolveSeq returns a resolve fake that yields a DIFFERENT programmed result on
// each successive call — used to drive the update path's pre-read and its post-write
// read-back independently (e.g. found on the pre-read, an authoritative 404 on the
// read-back).
func fipResolveSeq(t *testing.T, results ...fipResolveResult) vpcFloatingIPResolveFunc {
	t.Helper()
	calls := 0
	return func(ctx context.Context, id string) (*client.FloatingIP, bool, error) {
		if calls >= len(results) {
			t.Fatalf("resolve called %d time(s), only %d programmed", calls+1, len(results))
		}
		r := results[calls]
		calls++
		return r.fip, r.found, r.err
	}
}

// "must not be reached" sentinels: they fail the test if an orchestration calls
// them on a path where it must not (proving e.g. "no PATCH on a converged update"
// or "no read-back after a provision error").
func fipResolveFatal(t *testing.T) vpcFloatingIPResolveFunc {
	return func(ctx context.Context, id string) (*client.FloatingIP, bool, error) {
		t.Fatal("resolve must not be reached on this path")
		return nil, false, nil
	}
}

func fipUpdateFatal(t *testing.T) func(ctx context.Context, fipID, description string) (string, error) {
	return func(ctx context.Context, fipID, description string) (string, error) {
		t.Fatal("UpdateDescription must not be called on this path")
		return "", nil
	}
}

func fipWaitFatal(t *testing.T) vpcActivityWaitFunc {
	return func(ctx context.Context, activityID string) error {
		t.Fatal("wait must not be called on this path")
		return nil
	}
}

func fipWaitOK() vpcActivityWaitFunc {
	return func(ctx context.Context, activityID string) error { return nil }
}

// newFloatingIPCreateData builds a ResourceData for a fresh create: no id yet (the
// id is resolved from the async provision activity by the client and set by the
// provider). description is seeded only when non-empty.
func newFloatingIPCreateData(t *testing.T, description string) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCFloatingIP().Schema, map[string]interface{}{})
	if description != "" {
		if err := d.Set("description", description); err != nil {
			t.Fatalf("seeding description: %v", err)
		}
	}
	return d
}

// newFloatingIPState builds a ResourceData standing for an existing floating IP in
// the state, with its id and the (only) mutable field seeded.
func newFloatingIPState(t *testing.T) *schema.ResourceData {
	t.Helper()
	d := schema.TestResourceDataRaw(t, resourceVPCFloatingIP().Schema, map[string]interface{}{})
	d.SetId("fip-1")
	if err := d.Set("description", "seeded"); err != nil {
		t.Fatalf("seeding description: %v", err)
	}
	return d
}

func fipHasWarning(diags diag.Diagnostics) bool {
	for _, dg := range diags {
		if dg.Severity == diag.Warning {
			return true
		}
	}
	return false
}

// ---- create ---------------------------------------------------------------

// TestCreateVPCFloatingIPWith pins the async provision orchestration. The worst
// outcome is a BILLABLE ORPHAN (provisioned platform-side, absent from the state),
// so state safety drives every failure mode:
//
//   - a provision error -> FAIL, never SetId (nothing confirmed to track);
//   - the client returning an empty id with no error -> FAIL CLOSED (R-Q2 depth);
//   - a successful provision -> SetId BEFORE anything else; the description PATCH is
//     issued ONLY when configured, and its failure is a WARNING (never a taint over
//     a cosmetic label); the read-back uses readAfterWrite so it NEVER drops the
//     just-provisioned (billable) id on an eventually-consistent 404.
func TestCreateVPCFloatingIPWith(t *testing.T) {
	ctx := context.Background()

	t.Run("a provision error fails and never sets the id", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "")
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision:         func(ctx context.Context) (string, error) { return "", errors.New("boom") },
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
			resolve:           fipResolveFatal(t),
		})
		if !diags.HasError() {
			t.Fatal("a provision error must surface as a diagnostic")
		}
		if d.Id() != "" {
			t.Fatalf("a provision error must NEVER SetId, got %q", d.Id())
		}
	})

	// R-Q2 guard in depth: the client must never return ("", nil), but if a contract
	// regression did, the provider must FAIL CLOSED rather than SetId("") and silently
	// orphan a BILLABLE IP. A mutant that trusted the empty id reds here.
	t.Run("an empty id with no error fails closed and never sets the id", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "")
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision:         func(ctx context.Context) (string, error) { return "", nil },
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
			resolve:           fipResolveFatal(t),
		})
		if !diags.HasError() {
			t.Fatal("an empty id with no error must FAIL CLOSED (R-Q2 no-id fatal)")
		}
		if d.Id() != "" {
			t.Fatalf("a no-id provision must NEVER SetId, got %q", d.Id())
		}
	})

	t.Run("a successful provision WITHOUT a configured description sets the id and issues NO PATCH", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "") // no description -> no PATCH
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision:         func(ctx context.Context) (string, error) { return "fip-99", nil },
			updateDescription: fipUpdateFatal(t), // must NOT be called when no description is configured
			wait:              fipWaitFatal(t),
			resolve: fipResolve(&client.FloatingIP{
				ID: "fip-99", IPAddress: "198.51.100.7", Description: "api-assigned",
			}, true, nil),
		})
		if diags.HasError() {
			t.Fatalf("a successful provision must succeed, got: %v", diags)
		}
		if d.Id() != "fip-99" {
			t.Fatalf("the provisioned id must be set, got %q", d.Id())
		}
		for k, want := range map[string]string{"ip_address": "198.51.100.7", "description": "api-assigned"} {
			if got := d.Get(k).(string); got != want {
				t.Fatalf("post-provision %s = %q, want %q", k, got, want)
			}
		}
	})

	t.Run("a successful provision WITH a configured description issues the PATCH then reads back", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "my fip")
		var gotID, gotDesc, gotWaitedActivity string
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision: func(ctx context.Context) (string, error) { return "fip-99", nil },
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) {
				// Ordering invariant: the id MUST already be set before the PATCH runs,
				// else a panic/interruption mid-PATCH would orphan the billable IP.
				if d.Id() != "fip-99" {
					t.Fatalf("SetId must run BEFORE the description PATCH (never orphan); d.Id() = %q at PATCH time", d.Id())
				}
				gotID, gotDesc = fipID, description
				return "act-1", nil
			},
			wait: func(ctx context.Context, activityID string) error {
				gotWaitedActivity = activityID
				return nil
			},
			resolve: fipResolve(&client.FloatingIP{ID: "fip-99", IPAddress: "198.51.100.7", Description: "my fip"}, true, nil),
		})
		if diags.HasError() {
			t.Fatalf("a successful provision+patch must succeed, got: %v", diags)
		}
		if d.Id() != "fip-99" {
			t.Fatalf("the provisioned id must be set, got %q", d.Id())
		}
		if gotID != "fip-99" || gotDesc != "my fip" {
			t.Fatalf("the description PATCH must target (fip-99, %q), got (%q, %q)", "my fip", gotID, gotDesc)
		}
		if gotWaitedActivity != "act-1" {
			t.Fatalf("create must wait on the PATCH activity %q, got %q", "act-1", gotWaitedActivity)
		}
	})

	// §5: provision is the BILLABLE, critical op; the description is cosmetic. If the
	// PATCH fails after a successful provision, tainting would deprovision a perfectly
	// good billable IP over a label and re-provision a new one. So the failure is a
	// WARNING (no error), the id is KEPT, and the read-back still runs. A mutant that
	// returned an ERROR here (forcing a taint/recreate) reds on HasError().
	t.Run("provision OK + description PATCH failure is a WARNING (not an error) and keeps the id", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "my fip")
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision: func(ctx context.Context) (string, error) { return "fip-99", nil },
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) {
				return "", errors.New("patch boom")
			},
			wait:    fipWaitFatal(t), // update returned an error before any wait
			resolve: fipResolve(&client.FloatingIP{ID: "fip-99", IPAddress: "198.51.100.7", Description: "api-assigned"}, true, nil),
		})
		if diags.HasError() {
			t.Fatalf("a description PATCH failure must NOT fail the create (no taint over a cosmetic label), got error: %v", diags)
		}
		if !fipHasWarning(diags) {
			t.Fatal("a description PATCH failure must surface a WARNING (never silently swallowed)")
		}
		if d.Id() != "fip-99" {
			t.Fatalf("the provisioned id must be KEPT after a PATCH failure, got %q", d.Id())
		}
		// The warning must name the id (so the operator can find the billable IP to
		// reconcile), and the read-back must still have populated the computed attributes.
		var warnDetail string
		for _, dg := range diags {
			if dg.Severity == diag.Warning {
				warnDetail = dg.Detail
			}
		}
		if !strings.Contains(warnDetail, "fip-99") {
			t.Fatalf("the PATCH-failure warning must name the floating IP id for reconciliation, got %q", warnDetail)
		}
		if got := d.Get("ip_address").(string); got != "198.51.100.7" {
			t.Fatalf("the read-back must still populate computed attributes on the warning path, ip_address = %q", got)
		}
	})

	// ⭐ never-orphan: a just-provisioned IP not yet visible by id (authoritative 404)
	// is eventual consistency, NOT a deletion. The create path uses readAfterWrite, so
	// it FAILS CLOSED keeping the id. A mutant using readForRefresh on the create path
	// would SetId("") and orphan the BILLABLE IP -> reds on BOTH checks.
	t.Run("provision OK + read-back 404 FAILS CLOSED and keeps the id (never orphan)", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "")
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision:         func(ctx context.Context) (string, error) { return "fip-99", nil },
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
			resolve:           fipResolve(nil, false, nil), // authoritative 404 right after provision
		})
		if !diags.HasError() {
			t.Fatal("a just-provisioned IP absent on read-back must FAIL CLOSED (eventual consistency), not report a clean apply")
		}
		if d.Id() != "fip-99" {
			t.Fatalf("the just-provisioned id must be KEPT (never orphaned), got %q", d.Id())
		}
	})

	t.Run("provision OK + read-back error FAILS CLOSED and keeps the id", func(t *testing.T) {
		d := newFloatingIPCreateData(t, "")
		diags := createVPCFloatingIPWith(ctx, d, vpcFloatingIPCreateFuncs{
			provision:         func(ctx context.Context) (string, error) { return "fip-99", nil },
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
			resolve:           fipResolve(nil, false, errors.New("403 forbidden")), // not absence -> fail closed
		})
		if !diags.HasError() {
			t.Fatal("a read-back error must FAIL CLOSED, never report a clean apply")
		}
		if d.Id() != "fip-99" {
			t.Fatalf("the just-provisioned id must be kept, got %q", d.Id())
		}
	})
}

// ---- read -----------------------------------------------------------------

// TestReadVPCFloatingIPInto pins the read wiring. The overriding invariant: the
// resource is NEVER dropped on an inconclusive read. ResolveByID's tri-state is the
// whole story — an error (403/206/transport) is NOT absence; only found=false (an
// AUTHORITATIVE 404) is, and even that drops only in readForRefresh (in
// readAfterWrite it fails closed: eventual consistency, never a deletion).
func TestReadVPCFloatingIPInto(t *testing.T) {
	ctx := context.Background()

	t.Run("a resolve error keeps the id and fails closed", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := readVPCFloatingIPInto(ctx, d, fipResolve(nil, false, errors.New("403 forbidden")), floatingIPReadForRefresh)
		if !diags.HasError() {
			t.Fatal("a resolve error (forbidden/partial is not absence) must fail closed with a diagnostic")
		}
		if d.Id() != "fip-1" {
			t.Fatalf("id must be preserved on a resolve error, got %q", d.Id())
		}
	})

	t.Run("readForRefresh: an authoritative 404 DROPS the resource", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := readVPCFloatingIPInto(ctx, d, fipResolve(nil, false, nil), floatingIPReadForRefresh)
		if diags.HasError() {
			t.Fatalf("an authoritative 404 must drop cleanly on refresh, got: %v", diags)
		}
		if d.Id() != "" {
			t.Fatalf("a confirmed-absent floating IP must be dropped (SetId(\"\")), got id %q", d.Id())
		}
	})

	// R-N2 / never-orphan: the SAME authoritative 404 that DROPS in readForRefresh
	// must FAIL CLOSED (keep the id) in readAfterWrite. Kills the mutant that reuses
	// readForRefresh on the create/update path and orphans a fresh BILLABLE id.
	t.Run("readAfterWrite: an authoritative 404 FAILS CLOSED and keeps the id", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := readVPCFloatingIPInto(ctx, d, fipResolve(nil, false, nil), floatingIPReadAfterWrite)
		if !diags.HasError() {
			t.Fatal("readAfterWrite must FAIL CLOSED on a 404 (eventual consistency), never drop a just-written id")
		}
		if d.Id() != "fip-1" {
			t.Fatalf("readAfterWrite must keep the just-written id, got %q (a drop here would orphan the billable IP)", d.Id())
		}
	})

	t.Run("a successful read of a BOUND floating IP repopulates the full flatten keyset", func(t *testing.T) {
		d := newFloatingIPState(t)
		fip := &client.FloatingIP{
			ID:             "fip-1",
			IPAddress:      "198.51.100.7",
			Description:    "fresh",
			StaticIP:       &client.FloatingIPStaticIP{ID: "si-1", Address: "10.0.1.50"},
			VPC:            &client.BaseObject{ID: "vpc-1"},
			PrivateNetwork: &client.BaseObject{ID: "pn-1"},
		}
		diags := readVPCFloatingIPInto(ctx, d, fipResolve(fip, true, nil), floatingIPReadForRefresh)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		for k, want := range map[string]string{
			"ip_address":         "198.51.100.7",
			"description":        "fresh",
			"static_ip_id":       "si-1",
			"static_ip_address":  "10.0.1.50",
			"vpc_id":             "vpc-1",
			"private_network_id": "pn-1",
		} {
			if got := d.Get(k).(string); got != want {
				t.Fatalf("%s = %q, want %q", k, got, want)
			}
		}
	})

	t.Run("nil staticIp/vpc/privateNetwork (UNBOUND) flatten to empty strings without panic", func(t *testing.T) {
		d := newFloatingIPState(t)
		fip := &client.FloatingIP{ID: "fip-1", IPAddress: "198.51.100.7", Description: "fresh"} // all associations nil
		diags := readVPCFloatingIPInto(ctx, d, fipResolve(fip, true, nil), floatingIPReadForRefresh)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		for _, k := range []string{"static_ip_id", "static_ip_address", "vpc_id", "private_network_id"} {
			if got := d.Get(k).(string); got != "" {
				t.Fatalf("nil association %s must flatten to \"\", got %q", k, got)
			}
		}
	})
}

// ---- update ---------------------------------------------------------------

// TestUpdateVPCFloatingIPWith pins the description PATCH logic. Only the description
// is mutable. The desired value is reconciled against a FRESH LIVE read, never the
// state alone: an inconclusive read or an authoritative absence FAILS CLOSED (never
// PATCH on ambiguous evidence), a converged read issues NO PATCH, and a real change
// issues the async PATCH and waits.
func TestUpdateVPCFloatingIPWith(t *testing.T) {
	ctx := context.Background()

	t.Run("a resolve error fails closed and never PATCHes", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve:           fipResolve(nil, false, errors.New("403 forbidden")),
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
		})
		if !diags.HasError() {
			t.Fatal("an inconclusive pre-update read must fail closed, never PATCH on ambiguous evidence")
		}
	})

	t.Run("an authoritative 404 before update fails closed and never PATCHes", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve:           fipResolve(nil, false, nil),
			updateDescription: fipUpdateFatal(t),
			wait:              fipWaitFatal(t),
		})
		if !diags.HasError() {
			t.Fatal("a 404 before update must fail closed, never PATCH a floating IP that is not present")
		}
	})

	// Converged: the live description already equals the desired one -> NO PATCH. A
	// mutant that always PATCHed (a needless async write + activity wait) reds here.
	t.Run("a converged description issues NO PATCH", func(t *testing.T) {
		d := newFloatingIPState(t) // desired description == "seeded"
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve:           fipResolve(&client.FloatingIP{ID: "fip-1", Description: "seeded"}, true, nil),
			updateDescription: fipUpdateFatal(t), // must NOT be called when already converged
			wait:              fipWaitFatal(t),
		})
		if diags.HasError() {
			t.Fatalf("a converged update must succeed without a PATCH, got: %v", diags)
		}
	})

	t.Run("a changed description issues the PATCH and waits", func(t *testing.T) {
		d := newFloatingIPState(t)
		if err := d.Set("description", "renamed"); err != nil {
			t.Fatalf("setting desired description: %v", err)
		}
		var gotID, gotDesc, gotWaitedActivity string
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve: fipResolve(&client.FloatingIP{ID: "fip-1", Description: "seeded"}, true, nil),
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) {
				gotID, gotDesc = fipID, description
				return "act-7", nil
			},
			wait: func(ctx context.Context, activityID string) error { gotWaitedActivity = activityID; return nil },
		})
		if diags.HasError() {
			t.Fatalf("a changed description must apply, got: %v", diags)
		}
		if gotID != "fip-1" || gotDesc != "renamed" {
			t.Fatalf("the PATCH must target (fip-1, %q), got (%q, %q)", "renamed", gotID, gotDesc)
		}
		if gotWaitedActivity != "act-7" {
			t.Fatalf("update must wait on the PATCH activity %q, got %q", "act-7", gotWaitedActivity)
		}
	})

	t.Run("a PATCH error surfaces as a diagnostic", func(t *testing.T) {
		d := newFloatingIPState(t)
		if err := d.Set("description", "renamed"); err != nil {
			t.Fatalf("setting desired description: %v", err)
		}
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve: fipResolve(&client.FloatingIP{ID: "fip-1", Description: "seeded"}, true, nil),
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) {
				return "", errors.New("patch boom")
			},
			wait: fipWaitFatal(t),
		})
		if !diags.HasError() {
			t.Fatal("a PATCH error must surface as a diagnostic")
		}
	})

	t.Run("a PATCH activity wait failure surfaces as a diagnostic", func(t *testing.T) {
		d := newFloatingIPState(t)
		if err := d.Set("description", "renamed"); err != nil {
			t.Fatalf("setting desired description: %v", err)
		}
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve:           fipResolve(&client.FloatingIP{ID: "fip-1", Description: "seeded"}, true, nil),
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) { return "act-7", nil },
			wait:              func(ctx context.Context, activityID string) error { return errors.New("activity failed") },
		})
		if !diags.HasError() {
			t.Fatal("a failed PATCH activity must surface as a diagnostic, never a silent success")
		}
	})

	// never-orphan on the UPDATE tail: the post-PATCH read-back must use
	// readAfterWrite, so an eventually-consistent 404 right after a successful PATCH
	// KEEPS the id. A mutant wiring readForRefresh on the update tail would SetId("")
	// and orphan the billable IP -> reds here.
	t.Run("the post-PATCH read-back keeps the id on a 404 (readAfterWrite, never orphan)", func(t *testing.T) {
		d := newFloatingIPState(t)
		if err := d.Set("description", "renamed"); err != nil {
			t.Fatalf("setting desired description: %v", err)
		}
		diags := updateVPCFloatingIPWith(ctx, d, vpcFloatingIPUpdateFuncs{
			resolve: fipResolveSeq(t,
				fipResolveResult{fip: &client.FloatingIP{ID: "fip-1", Description: "seeded"}, found: true}, // pre-read drives the PATCH
				fipResolveResult{found: false}, // post-PATCH read-back: authoritative 404 (eventual consistency)
			),
			updateDescription: func(ctx context.Context, fipID, description string) (string, error) { return "act-7", nil },
			wait:              fipWaitOK(),
		})
		if !diags.HasError() {
			t.Fatal("a post-update 404 must FAIL CLOSED (eventual consistency), never drop the id")
		}
		if d.Id() != "fip-1" {
			t.Fatalf("the id must be KEPT after a post-update 404 (never orphan a billable IP), got %q", d.Id())
		}
	})
}

// ---- delete ---------------------------------------------------------------

// TestDeleteVPCFloatingIPWith pins the delete wiring. The gating + positive-404
// confirmation live in the client (DeprovisionUnbound, covered by the client
// lifecycle tests); this resource path must faithfully surface the outcome — a
// clean deprovision succeeds, and any deprovision error (e.g. a still-bound IP) is
// NEVER swallowed into a false success on a BILLABLE resource.
func TestDeleteVPCFloatingIPWith(t *testing.T) {
	ctx := context.Background()

	t.Run("a clean deprovision succeeds", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := deleteVPCFloatingIPWith(ctx, d, func(ctx context.Context, fipID string) error {
			if fipID != "fip-1" {
				t.Fatalf("deprovision must target the resource id, got %q", fipID)
			}
			return nil
		})
		if diags.HasError() {
			t.Fatalf("a clean deprovision must succeed, got: %v", diags)
		}
	})

	t.Run("a deprovision error (e.g. still bound) surfaces as a diagnostic", func(t *testing.T) {
		d := newFloatingIPState(t)
		diags := deleteVPCFloatingIPWith(ctx, d, func(ctx context.Context, fipID string) error {
			return errors.New(`floating IP "fip-1" is bound to static IP "si-1"; unbind or destroy the binding before deprovisioning`)
		})
		if !diags.HasError() {
			t.Fatal("a deprovision error must surface as a diagnostic, never a silent success on a billable resource")
		}
		if d.Id() != "fip-1" {
			t.Fatalf("a refused deprovision must KEEP the resource in state (never orphan the still-bound billable IP), got id %q", d.Id())
		}
	})
}

// ---- schema pins ----------------------------------------------------------

// TestVPCFloatingIPResourceSchemaKeyset pins that the resource schema declares
// EXACTLY the FlattenFloatingIP keyset and nothing else — a swap/drop of a flat
// field (or a stray schema key) goes RED here.
func TestVPCFloatingIPResourceSchemaKeyset(t *testing.T) {
	flat := map[string]bool{
		"id": true, "ip_address": true, "description": true,
		"static_ip_id": true, "static_ip_address": true,
		"vpc_id": true, "private_network_id": true,
	}

	schemaKeys := map[string]bool{}
	for k := range resourceVPCFloatingIP().Schema {
		schemaKeys[k] = true
	}

	if len(schemaKeys) != len(flat) {
		var sk, fk []string
		for k := range schemaKeys {
			sk = append(sk, k)
		}
		for k := range flat {
			fk = append(fk, k)
		}
		sort.Strings(sk)
		sort.Strings(fk)
		t.Fatalf("schema keyset must equal the FlattenFloatingIP keyset.\nschema:  %v\nflatten: %v", sk, fk)
	}
	for k := range flat {
		if !schemaKeys[k] {
			t.Fatalf("FlattenFloatingIP key %q is missing from the resource schema", k)
		}
	}
}

// TestVPCFloatingIPDescriptionSchema pins the DELIBERATE design choice that
// distinguishes the floating IP from the static IP: description is Optional +
// Computed with NO Default (the provision contract is POST {"count":1} and takes no
// description), and a ValidateFunc rejects an empty/whitespace value when one IS
// set. A mutant that adds a Default, makes it Required, or drops the ValidateFunc
// goes RED here.
func TestVPCFloatingIPDescriptionSchema(t *testing.T) {
	s, ok := resourceVPCFloatingIP().Schema["description"]
	if !ok {
		t.Fatal("description attribute is missing from the schema")
	}
	if s.Required {
		t.Fatal("description must be Optional, not Required (provision takes no description)")
	}
	if !s.Optional {
		t.Fatal("description must be Optional")
	}
	if !s.Computed {
		t.Fatal("description must be Computed (the API-assigned value is kept when omitted)")
	}
	if s.Default != nil {
		t.Fatalf("description must have NO Default (differs from static_ip; provision takes no description), got %v", s.Default)
	}
	if s.ValidateFunc == nil {
		t.Fatal("description must carry a ValidateFunc rejecting empty/whitespace when set")
	}
	for _, bad := range []string{"", "   ", "\t"} {
		if _, errs := s.ValidateFunc(bad, "description"); len(errs) == 0 {
			t.Fatalf("description must reject %q (empty/whitespace), got no error", bad)
		}
	}
	for _, good := range []string{"prod egress IP", "Managed by Terraform"} {
		if _, errs := s.ValidateFunc(good, "description"); len(errs) != 0 {
			t.Fatalf("description must accept %q, got %v", good, errs)
		}
	}
}

// TestVPCFloatingIPBindingAttributesAreReadOnly pins the C4 scope decision: the
// binding attributes are COMPUTED (read-only) — binding is NOT managed by this
// resource (the client's only deletion path refuses a bound IP). A mutant that made
// static_ip_id (or any binding field) settable goes RED here.
func TestVPCFloatingIPBindingAttributesAreReadOnly(t *testing.T) {
	for _, k := range []string{"static_ip_id", "static_ip_address", "vpc_id", "private_network_id", "ip_address"} {
		s, ok := resourceVPCFloatingIP().Schema[k]
		if !ok {
			t.Fatalf("%q attribute is missing from the schema", k)
		}
		if !s.Computed {
			t.Fatalf("%q must be Computed (read-only)", k)
		}
		if s.Optional || s.Required {
			t.Fatalf("%q must NOT be settable (binding is not managed by this resource)", k)
		}
	}
}

// TestVPCFloatingIPResourceTimeouts pins generous async timeouts on a BILLABLE
// resource: a premature timeout mid-provision/deprovision is exactly what must not
// happen. A mutant that dropped the Timeouts block goes RED here.
func TestVPCFloatingIPResourceTimeouts(t *testing.T) {
	r := resourceVPCFloatingIP()
	if r.Timeouts == nil {
		t.Fatal("the floating IP resource must declare Timeouts (async, billable)")
	}
	if r.Timeouts.Create == nil || r.Timeouts.Update == nil || r.Timeouts.Delete == nil {
		t.Fatal("Create/Update/Delete timeouts must all be set")
	}
}
