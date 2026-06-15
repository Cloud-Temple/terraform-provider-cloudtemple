package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// newRunToken returns a 128-bit hex token, mixed into every created resource's
// name so identities are collision-free across runs/workers: a stale orphan from
// a previous run can never be confused with — and wrongly deleted instead of — a
// fresh resource by the find-by-name teardown fallback. It FAILS rather than
// fall back to a constant: a non-unique identity on a destructive write cycle is
// unsafe, so the cycle aborts (creating nothing) if randomness is unavailable.
func newRunToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// computeLifecycleCycle is the end-to-end OpenIaaS compute WRITE business cycle:
// create a VM from a template, attach a user data disk (on an SR in the VM's
// pool), attach a user network adapter, then deprovision the whole stack
// leaves-first. It is the
// realization of the #316 TODO and the cycle to replay (with -runs/-concurrency)
// for a BOUNDED, breaker-guarded load probe ("how far does it hold").
//
// Every created resource has its teardown REGISTERED BEFORE the create (F3),
// keyed by a deterministic per-(iteration,worker) identity, so a created-but-
// unresolved resource is still swept; the deferred TeardownAll re-runs them
// idempotently (404-only) even if the explicit deprovision below already ran.
// Teardown order is leaves-first by construction (register VM, then disk, then
// adapter → LIFO removes adapter, then disk, then VM): a VM delete never cascades
// a user disk/adapter, so those must be gone first.
//
// v1 scope: lean and load-friendly — no power-on / driver wait / resize /
// relocate / VPC edge (those widen the failure surface and slow the cycle); they
// are deferred to a richer variant.
type computeLifecycleCycle struct {
	// tokenFunc overrides the run-identity generator (nil → newRunToken); used by
	// tests to exercise the identity-failure path.
	tokenFunc func() (string, error)
}

func (computeLifecycleCycle) Name() string { return "compute_lifecycle" }
func (computeLifecycleCycle) Kind() Kind   { return KindWrite }

// vmSize returns the VM create size for one dimension (CPU or memory): the larger
// of the floor and the template's own requirement. Deploying a template below its
// defined CPU/RAM is rejected by the order layer (MEMORY_CONSTRAINT_VIOLATION_ORDER),
// so the template's value must win when it exceeds the floor; the floor guards the
// case where the listing reports 0.
func vmSize(floor, fromTemplate int) int {
	if fromTemplate > floor {
		return fromTemplate
	}
	return floor
}

// openIaaSVMCreateReq builds the create request for the OpenIaaS lifecycle VM,
// sizing CPU/RAM at max(floor, template) so the deploy satisfies the order layer's
// "not below the template" constraint. Kept as a pure builder so a test can pin
// that the SELECTED template's sizing actually reaches the create request (not the
// bare floor constants).
// selectOpenIaaSDiskSR picks a storage repository usable for a data disk on a VM
// in the given pool/host. The disk-create rejects an SR the VM cannot reach
// ("could not find storage repository with uuid …"), so the SR must be in the
// VM's pool: a SHARED SR (reachable from any host in the pool) is preferred,
// otherwise a non-shared SR LOCAL to the VM's host. Only accessible, non-
// maintenance SRs with room for the disk qualify. Returns "" when none fits (the
// disk sub-step is then skipped — not a failure).
func selectOpenIaaSDiskSR(srs []*client.OpenIaaSStorageRepository, vmPoolID, vmHostID string, minFree int) string {
	var hostLocal string
	for _, sr := range srs {
		if sr == nil || sr.ID == "" || sr.MaintenanceMode || sr.Accessible == 0 || sr.FreeCapacity < minFree {
			continue
		}
		if sr.Shared && vmPoolID != "" && sr.Pool.ID == vmPoolID {
			return sr.ID // shared SR in the VM's pool — reachable from its host
		}
		if !sr.Shared && vmHostID != "" && sr.Host.ID == vmHostID && hostLocal == "" {
			hostLocal = sr.ID // fallback: a local SR on the VM's own host
		}
	}
	return hostLocal
}

func openIaaSVMCreateReq(name, templateID string, tmplCPU, tmplMem int) *client.CreateOpenIaasVirtualMachineRequest {
	return &client.CreateOpenIaasVirtualMachineRequest{
		Name:       name,
		TemplateID: templateID,
		CPU:        vmSize(clVMCPU, tmplCPU),
		Memory:     vmSize(clVMMemory, tmplMem),
	}
}

func (cyc computeLifecycleCycle) mkToken() (string, error) {
	if cyc.tokenFunc != nil {
		return cyc.tokenFunc()
	}
	return newRunToken()
}

const (
	// clVMCPU/clVMMemory are FLOORS: the VM is sized at max(floor, template's own
	// CPU/RAM), because the order layer rejects deploying a template with less than
	// its defined CPU/RAM (MEMORY_CONSTRAINT_VIOLATION_ORDER).
	clVMCPU    = 1
	clVMMemory = 1073741824 // 1 GiB, in bytes
	clDiskSize = 1073741824 // 1 GiB, in bytes
)

// resolveActivityResultID returns the single activity-state Result (the created
// resource id), or "" when the activity is nil or did not resolve to exactly one
// state — mirrors setIdFromActivityState (internal/provider/provider.go) so the
// harness reads the create id the same way the provider does.
func resolveActivityResultID(act *client.Activity) string {
	if act == nil || len(act.State) != 1 {
		return ""
	}
	for _, st := range act.State {
		return st.Result
	}
	return ""
}

func (cyc computeLifecycleCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// Collision-free identity: a 128-bit run token + full (Iteration, Worker)
	// integers (no byte() truncation). The find-by-name teardown fallback relies
	// on this being unique across runs and concurrent workers. If randomness is
	// unavailable we abort BEFORE creating anything (no non-unique identity). The
	// network adapter's MAC is left to the platform (omitted on create) so there
	// is no MAC-collision surface; the adapter teardown is scoped to THIS VM.
	var token string
	if r.op(cyc, "compute.openiaas.run_identity", func() error {
		t, e := cyc.mkToken()
		token = t
		return e
	}) != nil {
		// Identity generation failed: recorded as a FAILURE op (so the run exits
		// non-zero — observable), and NOTHING is created.
		return nil
	}
	name := fmt.Sprintf("ct-validate-%s-%d-%d", token, r.Iteration, r.Worker)

	oi := c.Compute().OpenIaaS()

	// --- PHASE 0: substrate (all read-discovered; SKIP write steps when absent,
	// never guess) ---
	var mmID string
	_ = r.op(cyc, "compute.openiaas.machine_managers.list", func() error {
		mms, err := oi.MachineManager().List(ctx)
		if err != nil {
			return err
		}
		for _, mm := range mms {
			if mm != nil && mm.ID != "" {
				mmID = mm.ID
				break
			}
		}
		return nil
	})

	// Every write/deprovision step, in order — used to skip-record the whole
	// cycle when there is no machine manager / template to deploy from.
	writeSteps := []string{
		"compute.openiaas.virtual_machine.create",
		"compute.openiaas.virtual_disk.create",
		"compute.openiaas.network_adapter.create",
		"compute.openiaas.virtual_machine.read",
		"compute.openiaas.network_adapter.delete",
		"compute.openiaas.virtual_disk.disconnect",
		"compute.openiaas.virtual_disk.delete",
		"compute.openiaas.virtual_machine.delete",
	}
	skipAll := func() {
		for _, ep := range writeSteps {
			r.skip(cyc, ep)
		}
	}

	if mmID == "" {
		r.skip(cyc, "compute.openiaas.templates.list")
		r.skip(cyc, "compute.openiaas.storage_repositories.list")
		r.skip(cyc, "compute.openiaas.networks.list")
		skipAll()
		return nil
	}

	var tmplID string
	var tmplCPU, tmplMem int
	_ = r.op(cyc, "compute.openiaas.templates.list", func() error {
		tmpls, err := oi.Template().List(ctx, &client.OpenIaaSTemplateFilter{MachineManagerId: mmID})
		if err != nil {
			return err
		}
		for _, t := range tmpls {
			if t != nil && t.ID != "" {
				tmplID, tmplCPU, tmplMem = t.ID, t.CPU, t.Memory
				break
			}
		}
		return nil
	})

	// A storage repository and a network are OPTIONAL: without them the disk and
	// adapter sub-steps are skipped, but the VM create+delete still run. The SR is
	// SELECTED later (after the VM is created), once the VM's pool/host is known —
	// the disk-create rejects an SR the VM cannot reach ("could not find storage
	// repository"), so a usable SR must be in the VM's pool, not just any SR.
	var srs []*client.OpenIaaSStorageRepository
	_ = r.op(cyc, "compute.openiaas.storage_repositories.list", func() error {
		var err error
		srs, err = oi.StorageRepository().List(ctx, &client.StorageRepositoryFilter{MachineManagerId: mmID})
		return err
	})
	var networkID string
	_ = r.op(cyc, "compute.openiaas.networks.list", func() error {
		nets, err := oi.Network().List(ctx, &client.OpenIaaSNetworkFilter{MachineManagerID: mmID})
		if err != nil {
			return err
		}
		for _, n := range nets {
			if n != nil && n.ID != "" {
				networkID = n.ID
				break
			}
		}
		return nil
	})

	if tmplID == "" {
		skipAll() // nothing to deploy from → not a failure
		return nil
	}

	// --- PHASE 1: provision the VM (teardown registered BEFORE the create) ---
	vmRef := &vmTeardownRef{Name: name, MachineManagerID: mmID}
	registerVMTeardown(r.Cleanup, computeVMSeam{c}, vmRef)
	var vmID string
	_ = r.op(cyc, "compute.openiaas.virtual_machine.create", func() error {
		activityID, err := oi.VirtualMachine().Create(ctx, openIaaSVMCreateReq(name, tmplID, tmplCPU, tmplMem))
		if err != nil {
			return err
		}
		act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		if werr != nil {
			return werr
		}
		vmID = resolveActivityResultID(act)
		return nil
	})
	if vmID == "" {
		// Create failed or its id did not resolve: the deferred VM teardown (by
		// name) sweeps it. Nothing further to provision/deprovision explicitly.
		for _, ep := range []string{
			"compute.openiaas.virtual_disk.create", "compute.openiaas.network_adapter.create",
			"compute.openiaas.virtual_machine.read",
			"compute.openiaas.network_adapter.delete", "compute.openiaas.virtual_disk.disconnect",
			"compute.openiaas.virtual_disk.delete", "compute.openiaas.virtual_machine.delete",
		} {
			r.skip(cyc, ep)
		}
		return nil
	}
	vmRef.ID, vmRef.Resolved = vmID, true

	// Read the created VM to learn its pool/host (doubles as the observational read
	// of the VM). The disk-create rejects an SR the VM cannot reach, so the SR is
	// selected against the VM's pool/host below — not blindly from the list.
	var vmPoolID, vmHostID string
	_ = r.op(cyc, "compute.openiaas.virtual_machine.read", func() error {
		vm, err := oi.VirtualMachine().Read(ctx, vmID)
		if err != nil {
			return err
		}
		if vm != nil {
			vmPoolID, vmHostID = vm.Pool.ID, vm.Host.ID
		}
		return nil
	})

	// --- PHASE 2: storage variation — attach a user data disk (optional) ---
	srID := selectOpenIaaSDiskSR(srs, vmPoolID, vmHostID, clDiskSize)
	var diskID string
	if srID != "" {
		diskRef := &diskTeardownRef{Name: name + "-data", VMID: vmID}
		registerVirtualDiskTeardown(r.Cleanup, computeVirtualDiskSeam{c}, diskRef)
		_ = r.op(cyc, "compute.openiaas.virtual_disk.create", func() error {
			activityID, err := oi.VirtualDisk().Create(ctx, &client.OpenIaaSVirtualDiskCreateRequest{
				Name:                name + "-data",
				Size:                clDiskSize,
				Mode:                "RW",
				StorageRepositoryID: srID,
				VirtualMachineID:    vmID,
			})
			if err != nil {
				return err
			}
			act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			if werr != nil {
				return werr
			}
			diskID = resolveActivityResultID(act)
			return nil
		})
		if diskID != "" {
			diskRef.ID, diskRef.Resolved = diskID, true
		}
	} else if len(srs) == 0 {
		// No storage repository at all on the tenant → nothing to exercise.
		r.skip(cyc, "compute.openiaas.virtual_disk.create")
	} else {
		// Candidate SRs EXIST but none is reachable for this VM (none shared in its
		// pool, none local to its host, or the VM read did not yield a pool/host).
		// Surface it as a recorded FAILURE rather than a silent skip: for a validation
		// tool, a green run that quietly never exercised the disk path is worse, and
		// the pool/host detail lets the reachability rule be refined.
		_ = r.op(cyc, "compute.openiaas.virtual_disk.create", func() error {
			return fmt.Errorf("no storage repository reachable for the VM (pool=%q host=%q; %d listed SR(s), none shared-in-pool nor host-local with >= %d bytes free)",
				vmPoolID, vmHostID, len(srs), clDiskSize)
		})
	}

	// --- PHASE 3: network connection — attach a user adapter (optional) ---------
	// The Create attaches the adapter to the network. No explicit Connect: bringing
	// the link up requires a RUNNING VM ("VM state is halted but should be running"),
	// and this lean cycle keeps the VM halted — the network attachment at provision
	// time is what we exercise (mirrors the VMware cycle).
	var adapterID string
	if networkID != "" {
		adapterRef := &adapterTeardownRef{VMID: vmID}
		registerNetworkAdapterTeardown(r.Cleanup, computeNetworkAdapterSeam{c}, adapterRef)
		_ = r.op(cyc, "compute.openiaas.network_adapter.create", func() error {
			activityID, err := oi.NetworkAdapter().Create(ctx, &client.CreateOpenIaasNetworkAdapterRequest{
				VirtualMachineID: vmID,
				NetworkID:        networkID,
				// MAC omitted: the platform assigns it (no MAC-collision surface).
			})
			if err != nil {
				return err
			}
			act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			if werr != nil {
				return werr
			}
			adapterID = resolveActivityResultID(act)
			return nil
		})
		if adapterID != "" {
			adapterRef.ID, adapterRef.Resolved = adapterID, true
		}
	} else {
		r.skip(cyc, "compute.openiaas.network_adapter.create")
	}

	// --- PHASE 4: deprovision (explicit + recorded; leaves-first). Overlaps the
	// deferred teardowns by design — idempotent (404-only), so a double delete is
	// not a false failure. Every op is breaker-gated, so once tripped these become
	// recorded skips and the deferred TeardownAll still sweeps everything. ---
	if adapterID != "" {
		_ = r.op(cyc, "compute.openiaas.network_adapter.delete", func() error {
			activityID, err := oi.NetworkAdapter().Delete(ctx, adapterID)
			if err != nil {
				return idempotentDeleteErr(err)
			}
			_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			return werr
		})
	} else {
		r.skip(cyc, "compute.openiaas.network_adapter.delete")
	}

	if diskID != "" {
		_ = r.op(cyc, "compute.openiaas.virtual_disk.disconnect", func() error {
			activityID, err := oi.VirtualDisk().Disconnect(ctx, diskID,
				&client.OpenIaaSVirtualDiskConnectionRequest{VirtualMachineID: vmID})
			if err != nil {
				return idempotentDeleteErr(err)
			}
			_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			return werr
		})
		_ = r.op(cyc, "compute.openiaas.virtual_disk.delete", func() error {
			activityID, err := oi.VirtualDisk().Delete(ctx, diskID)
			if err != nil {
				return idempotentDeleteErr(err)
			}
			_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			return werr
		})
	} else {
		r.skip(cyc, "compute.openiaas.virtual_disk.disconnect")
		r.skip(cyc, "compute.openiaas.virtual_disk.delete")
	}

	_ = r.op(cyc, "compute.openiaas.virtual_machine.delete", func() error {
		activityID, err := oi.VirtualMachine().Delete(ctx, vmID)
		if err != nil {
			return idempotentDeleteErr(err)
		}
		_, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		return werr
	})

	return nil
}
