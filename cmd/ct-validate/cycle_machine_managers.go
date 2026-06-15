package main

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// machineManagersCycle is a READ-ONLY probe of exactly the first two steps of the
// compute_lifecycle cycle: the local run-identity token, then the OpenIaaS
// machine_managers list. The list call is the SAME c.Compute().OpenIaaS().
// MachineManager().List(ctx) the vm/readonly cycles make — byte-identical through
// the real provider client (not curl). It exists to characterize, in isolation and
// under -runs/-concurrency, the intermittent 5xx on machine_managers.list (#315):
//
//	1  ok    compute.openiaas.run_identity            0 ms
//	2  FAIL  compute.openiaas.machine_managers.list   http_5xx
//
// No writes, no substrate discovery beyond this one list, no teardown.
type machineManagersCycle struct{}

func (machineManagersCycle) Name() string { return "machine_managers" }
func (machineManagersCycle) Kind() Kind   { return KindRead }

func (machineManagersCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// Step 1: run identity — a local 128-bit token, never an API call (mirrors the
	// compute_lifecycle cycle's first step so the output reads identically).
	_ = r.op(machineManagersCycle{}, "compute.openiaas.run_identity", func() error {
		_, err := newRunToken()
		return err
	})
	// Step 2: the exact machine_managers list call. Its error (incl. an http_5xx) is
	// recorded and categorized by r.op exactly as in the vm cycle.
	_ = r.op(machineManagersCycle{}, "compute.openiaas.machine_managers.list", func() error {
		_, err := c.Compute().OpenIaaS().MachineManager().List(ctx)
		return err
	})
	return nil
}
