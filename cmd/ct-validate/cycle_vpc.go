package main

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// silentWaiter is a WaiterOptions whose logger is a no-op: the harness records
// latency/outcome itself and does not want the client's per-poll chatter on
// stdout.
var silentWaiter = &client.WaiterOptions{Logger: func(string) {}}

// REBUILDING CONTRACT — opt-in only. This cycle exercises the /vpc/v1 API, which
// is being rebuilt for v1.9.0 (see internal/client/vpc.go): the client now speaks
// the new async contract, but the rebuild is not yet end-to-end validated. It
// stays QUARANTINED: excluded from the "all" selector and from the default
// read-only sweep, so a blanket `-cycles all -write` can never fire VPC writes
// against the still-evolving contract. It runs ONLY when named explicitly:
// `-cycles vpc -write` — which is how the C6 live end-to-end validation invokes it.
//
// vpcCycle drives a realistic VPC write business cycle:
//
//	provision floating IP -> set description -> confirm by read -> deprovision FIP
//	create custom static IP -> (if a spare unbound FIP exists) bind FIP ->
//	confirm by read -> unbind FIP -> delete static IP
//
// Every created resource is registered for teardown BEFORE the step that could
// lose it, so the never-orphan backstop holds even if a later step or the
// breaker aborts. When no spare FIP exists, the bind/unbind steps are recorded
// as skipped and only the create/delete sub-cycle runs.
//
// The floating-IP provision sub-cycle runs FIRST and is self-contained: it is
// independent of any private network (the static-IP sub-cycle below needs one),
// and it both provisions AND deprovisions a FRESH, billable public floating IP
// within the cycle. Because the provision body is count-only ({"count":1}) there
// is no deterministic pre-create key, so a provision whose id is lost before we
// resolve it is the irreducible orphan sub-window recovered only by a manual
// audit (see registerFloatingIPTeardown / floatingIPTeardownRef in
// cleanup_seams.go).
//
// This is the cycle the harness exists for: the 2026-06-15 incident was a VPC
// write loop amplifying an outage and orphaning static IPs. Here every write is
// bounded, breaker-gated, and teardown-backed.
type vpcCycle struct{}

func (vpcCycle) Name() string { return "vpc" }
func (vpcCycle) Kind() Kind   { return KindWrite }

// Quarantined excludes vpcCycle from the "all" selector (see cycle.go): the
// v1.9.0 /vpc/v1 rebuild is not yet end-to-end validated, so a blanket `-cycles
// all -write` must never fire VPC writes against it. It runs only when named
// explicitly: `-cycles vpc -write`. (TestBuildRegistryQuarantinesVPC pins this;
// lifting it is a deliberate end-of-rebuild step, not a C1 change.)
func (vpcCycle) Quarantined() bool { return true }

func (vc vpcCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// FIP provision/describe/confirm/deprovision sub-cycle (C4). Self-contained and
	// independent of any private network, so it runs FIRST — regardless of whether a
	// private network exists for the static-IP sub-cycle below. It provisions a
	// FRESH billable floating IP and deprovisions it within the cycle.
	vc.provisionFloatingIPSubCycle(ctx, c, r)

	pn, err := vc.pickPrivateNetwork(ctx, c, r)
	if err != nil || pn == nil {
		// pickPrivateNetwork already recorded the failure/skip.
		return err
	}

	// MAC unique per (iteration, worker) to avoid collisions across concurrent
	// or repeated runs. The 02:00:5e:f0:XX:YY range is locally-administered.
	mac := fmt.Sprintf("02:00:5e:f0:%02x:%02x", byte(r.Iteration), byte(r.Worker))

	// F3 + the async-teardown doctrine (mirrors registerVirtualDiskTeardown /
	// registerNetworkAdapterTeardown): register ONE ref-based teardown BEFORE the
	// create. Registered pre-POST, it covers process-death AND the "POST accepted
	// server-side but CreateStart errored before yielding an id" ambiguous window
	// (the static IP would otherwise orphan). The ref is filled as the cycle
	// advances: ActivityID/ID/Resolved on create, ExplicitlyDeleted after delete.
	ref := &staticIPTeardownRef{PrivateNetworkID: pn.ID, MAC: mac}
	registerStaticIPTeardown(r.Cleanup, vpcStaticIPSeam{c}, ref)

	var staticID string
	if err := r.op(vc, "vpc.static_ip.create", func() error {
		// CreateStart + WaitCreate (not the composed Create) so the cycle records
		// the create ACTIVITY id in the ref BEFORE waiting: if the wait then fails,
		// the orphan-window diagnostic can name the activity. This also exercises
		// live the exact split the provider (C3) will use.
		activityID, syncID, cerr := c.VPC().StaticIP().CreateStart(ctx, pn.ID, &client.CreateStaticIPRequest{
			MacAddress:          mac,
			ResourceDescription: "ct-validate",
		})
		if cerr != nil {
			return cerr
		}
		ref.ActivityID = activityID
		id := syncID
		if activityID != "" {
			waited, werr := c.VPC().StaticIP().WaitCreate(ctx, activityID, silentWaiter)
			if werr != nil {
				return fmt.Errorf("static IP create activity %s did not complete: %w", activityID, werr)
			}
			id = waited
		}
		staticID = id
		ref.ID = id
		ref.Resolved = true
		return nil
	}); err != nil || staticID == "" {
		// Create failed or yielded no id. ref.Resolved stays false, so the
		// pre-registered teardown sweeps a created-but-unresolved (or failed) static
		// IP via the strict by-MAC listing. Safe to return.
		return err
	}

	vc.bindSubCycle(ctx, c, r, staticID)

	// Explicit delete (recorded), overlapping the teardown by design: the cycle
	// deletes on the happy path; the teardown is the safety net if we never reach
	// here. The same-cycle absence proof is set ONLY on real success.
	vc.explicitDeleteStaticIP(ctx, c, r, ref)
	return nil
}

// explicitDeleteStaticIP runs the breaker-gated explicit delete and records STRICT
// same-cycle proof of absence into ref.ExplicitlyDeleted ONLY when the delete op
// actually RAN and succeeded. It mirrors computeLifecycleCycle.explicitDelete
// (cycle_compute_lifecycle.go): on a breaker skip r.op never invokes the closure,
// so the proof stays false and the deferred teardown fails closed on a later
// 403-on-absent (#303) instead of masking a possible orphan. The proof MUST be set
// from inside the closure — NEVER inferred from r.op's return value, which is nil
// on a breaker skip too.
func (vc vpcCycle) explicitDeleteStaticIP(ctx context.Context, c *client.Client, r *Run, ref *staticIPTeardownRef) {
	_ = r.op(vc, "vpc.static_ip.delete", func() error {
		activityID, derr := c.VPC().StaticIP().Delete(ctx, ref.ID)
		if derr != nil {
			return derr
		}
		if activityID != "" {
			if _, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
				return werr
			}
		}
		ref.ExplicitlyDeleted = true
		return nil
	})
}

// pickPrivateNetwork lists private networks and returns the one named "LAN", or
// the first one otherwise. It records the listing endpoint and skips the cycle
// (no error) when there is no private network to work in.
func (vc vpcCycle) pickPrivateNetwork(ctx context.Context, c *client.Client, r *Run) (*client.PrivateNetwork, error) {
	var pns []*client.PrivateNetwork
	if err := r.op(vc, "vpc.private_networks.list", func() error {
		var lerr error
		pns, lerr = c.VPC().PrivateNetwork().List(ctx, nil)
		return lerr
	}); err != nil {
		return nil, err
	}
	if len(pns) == 0 {
		r.skip(vc, "vpc.static_ip.create")
		r.skip(vc, "vpc.static_ip.delete")
		r.skip(vc, "vpc.floating_ip.bind")
		r.skip(vc, "vpc.floating_ip.unbind")
		return nil, nil
	}
	for _, pn := range pns {
		if pn.Name != nil && *pn.Name == "LAN" {
			return pn, nil
		}
	}
	return pns[0], nil
}

// bindSubCycle finds a spare UNBOUND floating IP, binds it to the static IP,
// confirms the binding by read, then unbinds it. When no spare FIP exists, the
// bind/unbind/confirm steps are recorded as skipped.
func (vc vpcCycle) bindSubCycle(ctx context.Context, c *client.Client, r *Run, staticID string) {
	var fips []*client.FloatingIP
	if err := r.op(vc, "vpc.floating_ips.list", func() error {
		var lerr error
		fips, lerr = c.VPC().FloatingIP().List(ctx, nil)
		return lerr
	}); err != nil {
		// Listing failed: cannot safely pick a FIP, so skip the bind steps.
		r.skip(vc, "vpc.floating_ip.bind")
		r.skip(vc, "vpc.floating_ip.unbind")
		return
	}

	var spare *client.FloatingIP
	for _, fip := range fips {
		if fip != nil && fip.StaticIP == nil {
			spare = fip
			break
		}
	}
	if spare == nil {
		r.skip(vc, "vpc.floating_ip.bind")
		r.skip(vc, "vpc.floating_ip.unbind")
		return
	}

	// F3: register the unbind teardown BEFORE the bind, keyed by the
	// deterministic (fip, static) pair. A bind whose activity completes but whose
	// confirmation is then lost (abort/panic between Bind and the registration)
	// must still release our FIP. The teardown is idempotent (already-unbound →
	// success), so registering it even if the bind ultimately fails is harmless.
	registerFIPUnbindTeardown(r.Cleanup, vpcFIPBindSeam{c}, spare.ID, staticID)

	if err := r.op(vc, "vpc.floating_ip.bind", func() error {
		activityID, berr := c.VPC().FloatingIP().Bind(ctx, spare.ID, staticID)
		if berr != nil {
			return berr
		}
		if activityID != "" {
			if _, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
				return werr
			}
		}
		return nil
	}); err != nil {
		// Bind failed; the idempotent unbind teardown registered above is a
		// best-effort no-op if the bind never took effect.
		r.skip(vc, "vpc.floating_ip.unbind")
		return
	}

	// Confirm the binding took effect (positive same-pair corroboration).
	_ = r.op(vc, "vpc.floating_ip.confirm_bound", func() error {
		state, cerr := c.VPC().FloatingIP().CorroborateBinding(ctx, spare.ID, staticID)
		if cerr != nil {
			return cerr
		}
		if state != client.FloatingIPBindingBoundToTarget {
			return fmt.Errorf("floating IP %s not confirmed bound to %s (state=%d)", spare.ID, staticID, state)
		}
		return nil
	})

	// Explicit unbind on the happy path (recorded). Overlaps with the
	// registered teardown by design.
	_ = r.op(vc, "vpc.floating_ip.unbind", func() error {
		activityID, uerr := c.VPC().FloatingIP().Unbind(ctx, spare.ID, staticID)
		if uerr != nil {
			return uerr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		return werr
	})
}

// provisionFloatingIPSubCycle provisions a FRESH, billable public floating IP,
// PATCHes its description to the "ct-validate" marker, confirms the marker by a
// by-id read, then deprovisions it. It is self-contained (needs no private
// network) and is the C4 lifecycle exercised live.
//
// F3 + the async-teardown doctrine: register ONE ref-based teardown BEFORE the
// provision. Unlike the static-IP ref there is NO deterministic pre-create key —
// the provision body is count-only ({"count":1}) — so the teardown can act only
// on the RESOLVED id; a provision that is accepted server-side but whose id we
// then lose is the irreducible orphan sub-window left to a documented manual
// audit (registerFloatingIPTeardown is a no-op when the ref is unresolved).
//
// The split is ProvisionStart + WaitProvision (not the composed Provision) so the
// cycle records the provision ACTIVITY id in the ref BEFORE waiting: if the wait
// then fails, the orphan-window diagnostic can name the activity. This also
// exercises live the exact split the provider (phase 3) will use.
func (vc vpcCycle) provisionFloatingIPSubCycle(ctx context.Context, c *client.Client, r *Run) {
	ref := &floatingIPTeardownRef{}
	registerFloatingIPTeardown(r.Cleanup, vpcFIPDeprovisionSeam{c}, ref)

	if err := r.op(vc, "vpc.floating_ip.provision", func() error {
		activityID, syncID, perr := c.VPC().FloatingIP().ProvisionStart(ctx)
		if perr != nil {
			return perr
		}
		ref.ActivityID = activityID
		id := syncID
		if activityID != "" {
			waited, werr := c.VPC().FloatingIP().WaitProvision(ctx, activityID, silentWaiter)
			if werr != nil {
				return fmt.Errorf("floating IP provision activity %s did not complete: %w", activityID, werr)
			}
			id = waited
		}
		ref.ID = id
		ref.Resolved = true
		return nil
	}); err != nil || !ref.Resolved || ref.ID == "" {
		// Provision failed, was breaker-skipped, or yielded no id. ref.Resolved stays
		// false, so the pre-registered teardown is a no-op (no key to sweep). Record
		// the remaining steps as skipped and stop — there is no FIP to describe or
		// deprovision.
		r.skip(vc, "vpc.floating_ip.describe")
		r.skip(vc, "vpc.floating_ip.confirm_described")
		r.skip(vc, "vpc.floating_ip.deprovision")
		return
	}

	// PATCH the description to a known marker. The count-only provision cannot carry
	// a description, so create-time convergence is a separate PATCH — exactly what
	// the provider Create will do in phase 3. Wait only when the PATCH yields an
	// activity handle (UpdateDescription fails closed on 202/no-Location).
	if err := r.op(vc, "vpc.floating_ip.describe", func() error {
		activityID, uerr := c.VPC().FloatingIP().UpdateDescription(ctx, ref.ID, "ct-validate")
		if uerr != nil {
			return uerr
		}
		if activityID != "" {
			if _, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter); werr != nil {
				return werr
			}
		}
		return nil
	}); err != nil {
		// Description PATCH failed; the FIP still exists. Confirm is moot, but the
		// billable resource MUST still be released — fall through to deprovision.
		r.skip(vc, "vpc.floating_ip.confirm_described")
		vc.explicitDeprovisionFloatingIP(ctx, c, r, ref)
		return
	}

	// Confirm the description took effect (positive by-id read-back corroboration).
	_ = r.op(vc, "vpc.floating_ip.confirm_described", func() error {
		fip, found, cerr := c.VPC().FloatingIP().ResolveByID(ctx, ref.ID)
		if cerr != nil {
			return cerr
		}
		if !found {
			return fmt.Errorf("floating IP %s not found on read-back after provision", ref.ID)
		}
		if fip.Description != "ct-validate" {
			return fmt.Errorf("floating IP %s description not confirmed (got %q, want %q)", ref.ID, fip.Description, "ct-validate")
		}
		return nil
	})

	// Explicit deprovision on the happy path (recorded), overlapping the registered
	// teardown by design: the cycle deprovisions here; the teardown is the safety net
	// if we never reach this point. Same-cycle absence proof is set ONLY on success.
	vc.explicitDeprovisionFloatingIP(ctx, c, r, ref)
}

// explicitDeprovisionFloatingIP runs the breaker-gated explicit deprovision and
// records STRICT same-cycle proof into ref.ExplicitlyDeleted ONLY when the op
// actually RAN and succeeded. It mirrors explicitDeleteStaticIP: on a breaker skip
// r.op never invokes the closure, so the proof stays false and the deferred
// teardown re-runs (a safe no-op via DeprovisionUnbound's by-id 404 → success)
// instead of masking a possible orphan. The proof MUST be set from inside the
// closure — NEVER inferred from r.op's return value, which is nil on a skip too.
func (vc vpcCycle) explicitDeprovisionFloatingIP(ctx context.Context, c *client.Client, r *Run, ref *floatingIPTeardownRef) {
	_ = r.op(vc, "vpc.floating_ip.deprovision", func() error {
		if derr := c.VPC().FloatingIP().DeprovisionUnbound(ctx, ref.ID, silentWaiter); derr != nil {
			return derr
		}
		ref.ExplicitlyDeleted = true
		return nil
	})
}
