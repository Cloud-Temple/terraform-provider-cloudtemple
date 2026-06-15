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

// vpcCycle drives a realistic VPC write business cycle:
//
//	create custom static IP -> (if a spare unbound FIP exists) bind FIP ->
//	confirm by read -> unbind FIP -> delete static IP
//
// Every created resource is registered for teardown BEFORE the step that could
// lose it, so the never-orphan backstop holds even if a later step or the
// breaker aborts. When no spare FIP exists, the bind/unbind steps are recorded
// as skipped and only the create/delete sub-cycle runs.
//
// This is the cycle the harness exists for: the 2026-06-15 incident was a VPC
// write loop amplifying an outage and orphaning static IPs. Here every write is
// bounded, breaker-gated, and teardown-backed.
type vpcCycle struct{}

func (vpcCycle) Name() string { return "vpc" }
func (vpcCycle) Kind() Kind   { return KindWrite }

func (vc vpcCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	pn, err := vc.pickPrivateNetwork(ctx, c, r)
	if err != nil || pn == nil {
		// pickPrivateNetwork already recorded the failure/skip.
		return err
	}

	// MAC unique per (iteration, worker) to avoid collisions across concurrent
	// or repeated runs. The 02:00:5e:f0:XX:YY range is locally-administered.
	mac := fmt.Sprintf("02:00:5e:f0:%02x:%02x", byte(r.Iteration), byte(r.Worker))

	var staticID string
	if err := r.op(vc, "vpc.static_ip.create", func() error {
		id, cerr := c.VPC().StaticIP().Create(ctx, pn.ID, &client.CreateStaticIPRequest{
			MacAddress:          mac,
			ResourceDescription: "ct-validate",
		})
		if cerr != nil {
			return cerr
		}
		staticID = id
		return nil
	}); err != nil || staticID == "" {
		// Either the create failed (recorded) or returned no id; nothing to
		// tear down (the client only returns a non-empty id when the resource
		// is confirmed to exist).
		return err
	}

	// Register teardown IMMEDIATELY: from here on, an abort must still delete
	// this static IP. Teardown is best-effort with a bounded transient-retry.
	r.Cleanup.Register(fmt.Sprintf("vpc.static_ip %s", staticID), func(tctx context.Context) error {
		activityID, derr := c.VPC().StaticIP().Delete(tctx, staticID)
		if derr != nil {
			return derr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(tctx, activityID, silentWaiter)
		return werr
	})

	vc.bindSubCycle(ctx, c, r, staticID)

	// Explicit delete step (recorded). It overlaps with the registered
	// teardown intentionally: the cycle deletes on the happy path; the
	// teardown is the safety net if we never reach here.
	_ = r.op(vc, "vpc.static_ip.delete", func() error {
		activityID, derr := c.VPC().StaticIP().Delete(ctx, staticID)
		if derr != nil {
			return derr
		}
		if activityID == "" {
			return nil
		}
		_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		return werr
	})
	return nil
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

	bound := false
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
		bound = true
		return nil
	}); err != nil {
		// Bind failed; nothing to unbind. The static IP teardown is already
		// registered.
		r.skip(vc, "vpc.floating_ip.unbind")
		return
	}

	if bound {
		// Register the unbind teardown BEFORE doing anything else with the
		// binding, so an abort still releases our FIP (LIFO: unbind runs before
		// the static-IP delete).
		r.Cleanup.Register(fmt.Sprintf("vpc.floating_ip unbind %s<-%s", spare.ID, staticID), func(tctx context.Context) error {
			activityID, uerr := c.VPC().FloatingIP().Unbind(tctx, spare.ID, staticID)
			if uerr != nil {
				return uerr
			}
			if activityID == "" {
				return nil
			}
			_, werr := c.Activity().WaitForCompletion(tctx, activityID, silentWaiter)
			return werr
		})
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
