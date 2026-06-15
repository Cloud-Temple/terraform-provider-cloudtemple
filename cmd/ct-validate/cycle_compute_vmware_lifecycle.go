package main

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// computeVMwareLifecycleCycle is the end-to-end VMware (vCenter) compute WRITE
// business cycle, the sibling of computeLifecycleCycle (OpenIaaS): create a VM
// from scratch on a discovered datacenter/host-cluster/datastore, attach a user
// data disk, attach a user network adapter, read the assembled VM, then
// deprovision the whole stack leaves-first.
//
// It is structurally IDENTICAL to the OpenIaaS cycle (register-before-create F3,
// LIFO leaves-first teardown, run-unique 128-bit identity, find-by-name fail-
// closed, 404-only idempotent deletes) so the two platforms get the SAME safety
// guarantees. Only the client surface differs:
//   - VMware mutations are async and resolve the created id from the activity's
//     CONCERNED ITEMS (setIdFromActivityConcernedItems, internal/provider/
//     provider.go) — NOT from a single State Result as OpenIaaS does. Hence the
//     dedicated resolveActivityConcernedItemID below.
//   - a VMware VM is created from substrate (datacenter + host + datastore +
//     guest-OS moref), not from a template id.
//   - a VMware data disk delete removes it outright (no separate disconnect step
//     like the OpenIaaS shared-disk model).
//
// v1 scope: lean and load-friendly — no power-on / guest customization / resize /
// relocate / backup SLA (those widen the failure surface and slow the cycle).
// The VM is created powered-off, so the explicit delete needs no power cycle; the
// deferred teardown still powers off best-effort as a belt-and-suspenders.
type computeVMwareLifecycleCycle struct {
	// tokenFunc overrides the run-identity generator (nil → newRunToken); used by
	// tests to exercise the identity-failure path.
	tokenFunc func() (string, error)
}

func (computeVMwareLifecycleCycle) Name() string { return "compute_vmware_lifecycle" }
func (computeVMwareLifecycleCycle) Kind() Kind   { return KindWrite }

func (cyc computeVMwareLifecycleCycle) mkToken() (string, error) {
	if cyc.tokenFunc != nil {
		return cyc.tokenFunc()
	}
	return newRunToken()
}

const (
	vmwVMCPU       = 1
	vmwVMMemory    = 1073741824  // 1 GiB, in bytes
	vmwDiskSize    = 10737418240 // 10 GiB, in bytes; "dynamic" provisioning is thin so the real footprint is ~0
	vmwAdapterType = "VMXNET3"   // a modern adapter type accepted by the common guest OSes
	vmwDiskProvKey = "dynamic"   // thin provisioning (provisioning_type enum: dynamic|staticImmediate|staticDiffered)
	vmwDiskMode    = "persistent"
)

// resolveActivityConcernedItemID returns the id of the created resource from a
// completed VMware create activity: the concerned item whose Type matches
// expectedType. It is FAIL-CLOSED — it requires EXACTLY ONE such item and errors
// otherwise:
//   - a nil activity, or no concerned item of the type, means the create
//     succeeded server-side but its id could not be resolved → the caller must
//     surface this as a FAILURE (observable, run exits non-zero) rather than
//     record a misleading OK; the teardown (registered before the create) still
//     sweeps the created resource by name (VM) or VM-scope (disk/adapter);
//   - MORE THAN ONE matching item is ambiguous (a stale/duplicate concerned item)
//     → refuse to guess which is ours and error, so the caller never deletes a
//     possibly-wrong resource by a mis-resolved id (it falls back to the safe
//     VM-scoped delete-all instead).
//
// This is deliberately STRICTER than the provider's setIdFromActivityConcernedItems
// (which takes the first match): a destructive load cycle must not act on an
// ambiguous or unverifiable id.
func resolveActivityConcernedItemID(act *client.Activity, expectedType string) (string, error) {
	if act == nil {
		return "", fmt.Errorf("create activity is nil: cannot resolve the %s id", expectedType)
	}
	var found string
	for _, ci := range act.ConcernedItems {
		if ci.Type == expectedType && ci.ID != "" {
			if found != "" {
				return "", fmt.Errorf("ambiguous create activity: more than one %s concerned item", expectedType)
			}
			found = ci.ID
		}
	}
	if found == "" {
		return "", fmt.Errorf("create activity carried no %s concerned item: created id unresolved", expectedType)
	}
	return found, nil
}

func (cyc computeVMwareLifecycleCycle) Run(ctx context.Context, c *client.Client, r *Run) error {
	// Collision-free identity, same contract as the OpenIaaS cycle: abort BEFORE
	// creating anything if randomness is unavailable (no non-unique identity on a
	// destructive write cycle). Recorded as a FAILURE op so the run exits non-zero.
	var token string
	if r.op(cyc, "compute.vmware.run_identity", func() error {
		t, e := cyc.mkToken()
		token = t
		return e
	}) != nil {
		return nil
	}
	name := fmt.Sprintf("ct-validate-%s-%d-%d", token, r.Iteration, r.Worker)

	cm := c.Compute()

	// Every write/deprovision step, in order — used to skip-record the whole cycle
	// when the substrate is absent (VMware not on this tenant, or no usable
	// datacenter/host/datastore/guest-OS).
	writeSteps := []string{
		"compute.vmware.virtual_machine.create",
		"compute.vmware.virtual_disk.create",
		"compute.vmware.network_adapter.create",
		"compute.vmware.virtual_machine.read",
		"compute.vmware.network_adapter.delete",
		"compute.vmware.virtual_disk.delete",
		"compute.vmware.virtual_machine.delete",
	}
	skipAll := func() {
		for _, ep := range writeSteps {
			r.skip(cyc, ep)
		}
	}

	// --- PHASE 0: substrate (all read-discovered; never guess) ------------------
	// Probe the datacenter list once: a 4xx means VMware is not available/usable on
	// this tenant (e.g. an OpenIaaS-only tenant), so SKIP the whole cycle rather
	// than emit a false "squeak" per write step. A 5xx/timeout still surfaces as a
	// real failure on the probe (and trips the breaker).
	var dcID, mmID string
	dcErr := r.op(cyc, "compute.vmware.virtual_datacenters.list", func() error {
		dcs, err := cm.VirtualDatacenter().List(ctx, nil)
		if err != nil {
			return err
		}
		for _, dc := range dcs {
			if dc != nil && dc.ID != "" {
				dcID = dc.ID
				mmID = dc.MachineManager.ID // the vCenter (machineManagerId) this datacenter belongs to
				break
			}
		}
		return nil
	})
	if categorize(dcErr) == CategoryHTTP4xx || dcID == "" {
		r.skip(cyc, "compute.vmware.host_clusters.list")
		r.skip(cyc, "compute.vmware.datastores.list")
		r.skip(cyc, "compute.vmware.networks.list")
		r.skip(cyc, "compute.vmware.guest_operating_systems.list")
		skipAll()
		return nil
	}

	// Placement: a HOST CLUSTER. The provider makes host_cluster_id mandatory for
	// every VMware create path (an individual host_id is optional), so the cluster
	// is the required placement unit; we let the cluster choose the host. Every
	// substrate list is scoped by both machineManagerId (the vCenter) and
	// datacenterId — the documented VMware id-chaining tree.
	var hostClusterID string
	_ = r.op(cyc, "compute.vmware.host_clusters.list", func() error {
		hcs, err := cm.HostCluster().List(ctx, &client.HostClusterFilter{MachineManagerId: mmID, DatacenterId: dcID})
		if err != nil {
			return err
		}
		for _, hc := range hcs {
			if hc != nil && hc.ID != "" {
				hostClusterID = hc.ID
				break
			}
		}
		return nil
	})

	var datastoreID string
	_ = r.op(cyc, "compute.vmware.datastores.list", func() error {
		dss, err := cm.Datastore().List(ctx, &client.DatastoreFilter{MachineManagerId: mmID, DatacenterId: dcID})
		if err != nil {
			return err
		}
		for _, ds := range dss {
			// Only an accessible, non-maintenance datastore with room for the disk.
			if ds != nil && ds.ID != "" && !ds.MaintenanceMode && ds.Accessible != 0 && ds.FreeCapacity >= vmwDiskSize {
				datastoreID = ds.ID
				break
			}
		}
		return nil
	})

	// A network is OPTIONAL: without one the adapter sub-steps are skipped, but the
	// VM create+disk+delete still run.
	var networkID string
	_ = r.op(cyc, "compute.vmware.networks.list", func() error {
		nets, err := cm.Network().List(ctx, &client.NetworkFilter{MachineManagerId: mmID, DatacenterId: dcID})
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

	// Guest-OS moref needs a host-cluster (or host) scope.
	var osMoref string
	if hostClusterID != "" {
		_ = r.op(cyc, "compute.vmware.guest_operating_systems.list", func() error {
			oses, err := cm.GuestOperatingSystem().List(ctx, &client.GuestOperatingSystemFilter{HostClusterID: hostClusterID})
			if err != nil {
				return err
			}
			for _, os := range oses {
				if os != nil && os.Moref != "" {
					osMoref = os.Moref
					break
				}
			}
			return nil
		})
	} else {
		r.skip(cyc, "compute.vmware.guest_operating_systems.list")
	}

	// A VM create needs datacenter + host cluster + datastore + guest-OS moref. Any
	// missing → nothing to deploy from → skip the write block (not a failure).
	if hostClusterID == "" || datastoreID == "" || osMoref == "" {
		skipAll()
		return nil
	}

	// --- PHASE 1: provision the VM (teardown registered BEFORE the create) ------
	vmRef := &vmwareVMTeardownRef{Name: name, DatacenterID: dcID}
	registerVMwareVMTeardown(r.Cleanup, computeVMwareVMSeam{c}, vmRef)
	var vmID string
	_ = r.op(cyc, "compute.vmware.virtual_machine.create", func() error {
		activityID, err := cm.VirtualMachine().Create(ctx, &client.CreateVirtualMachineRequest{
			Name:                      name,
			DatacenterId:              dcID,
			HostClusterId:             hostClusterID,
			DatastoreId:               datastoreID,
			CPU:                       vmwVMCPU,
			Memory:                    vmwVMMemory,
			GuestOperatingSystemMoref: osMoref,
		})
		if err != nil {
			return err
		}
		act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		if werr != nil {
			return werr
		}
		id, rerr := resolveActivityConcernedItemID(act, "virtual_machine")
		if rerr != nil {
			return rerr // observable failure; the pre-registered VM teardown sweeps it by name
		}
		vmID = id
		return nil
	})
	if vmID == "" {
		// Create failed or its id did not resolve: the deferred VM teardown (by
		// name) sweeps it. Nothing further to provision/deprovision explicitly.
		for _, ep := range []string{
			"compute.vmware.virtual_disk.create", "compute.vmware.network_adapter.create",
			"compute.vmware.virtual_machine.read",
			"compute.vmware.network_adapter.delete", "compute.vmware.virtual_disk.delete",
			"compute.vmware.virtual_machine.delete",
		} {
			r.skip(cyc, ep)
		}
		return nil
	}
	vmRef.ID, vmRef.Resolved = vmID, true

	// --- PHASE 2: storage variation — attach a user data disk ------------------
	// The disk carries no caller-chosen name (the VMware API assigns one), so the
	// teardown recovers it by the VM it is attached to, not by name.
	diskRef := &vmwareDiskTeardownRef{VMID: vmID}
	registerVMwareDiskTeardown(r.Cleanup, computeVMwareDiskSeam{c}, diskRef)
	var diskID string
	_ = r.op(cyc, "compute.vmware.virtual_disk.create", func() error {
		activityID, err := cm.VirtualDisk().Create(ctx, &client.CreateVirtualDiskRequest{
			ProvisioningType: vmwDiskProvKey,
			DiskMode:         vmwDiskMode,
			Capacity:         vmwDiskSize,
			VirtualMachineId: vmID,
			DatastoreId:      datastoreID,
		})
		if err != nil {
			return err
		}
		act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
		if werr != nil {
			return werr
		}
		id, rerr := resolveActivityConcernedItemID(act, "virtual_disk")
		if rerr != nil {
			// Observable failure; diskID stays "" so the deprovision falls back to
			// the VM-scoped delete-all, which removes the created disk regardless.
			return rerr
		}
		diskID = id
		return nil
	})
	if diskID != "" {
		diskRef.ID, diskRef.Resolved = diskID, true
	}

	// --- PHASE 3: network connection — attach a user adapter (optional) ---------
	// On VMware the Create already attaches the adapter to the network (unlike the
	// OpenIaaS flow, which needs a separate Connect). Connect/Disconnect only flip
	// the live link state and are needlessly fragile on a fresh VM, so the cycle
	// stops at Create — the network connection is realized at provision time.
	var adapterID string
	if networkID != "" {
		adapterRef := &vmwareAdapterTeardownRef{VMID: vmID}
		registerVMwareNetworkAdapterTeardown(r.Cleanup, computeVMwareAdapterSeam{c}, adapterRef)
		_ = r.op(cyc, "compute.vmware.network_adapter.create", func() error {
			activityID, err := cm.NetworkAdapter().Create(ctx, &client.CreateNetworkAdapterRequest{
				VirtualMachineId: vmID,
				NetworkId:        networkID,
				Type:             vmwAdapterType,
				// MAC omitted: the platform assigns it (no MAC-collision surface).
			})
			if err != nil {
				return err
			}
			act, werr := c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
			if werr != nil {
				return werr
			}
			id, rerr := resolveActivityConcernedItemID(act, "network_adapter")
			if rerr != nil {
				// Observable failure; adapterID stays "" so the deprovision falls back
				// to the VM-scoped delete-all, which removes the created adapter.
				return rerr
			}
			adapterID = id
			return nil
		})
		if adapterID != "" {
			adapterRef.ID, adapterRef.Resolved = adapterID, true
		}
	} else {
		r.skip(cyc, "compute.vmware.network_adapter.create")
	}

	// Observational read of the assembled VM.
	_ = r.op(cyc, "compute.vmware.virtual_machine.read", func() error {
		_, err := cm.VirtualMachine().Read(ctx, vmID)
		return err
	})

	// --- PHASE 4: deprovision (explicit + recorded; leaves-first, WHILE THE VM
	// STILL EXISTS) ---
	vmwareDeprovision(ctx, r, cyc, vmID, networkID, adapterID, diskID,
		computeVMwareAdapterSeam{c}, computeVMwareDiskSeam{c}, computeVMwareVMSeam{c})
	return nil
}

// deleteVMwareLeaves removes a VM-scoped leaf (disk or adapter): the resolved id
// when known, else EVERY such leaf the VM still carries (VM-scoped delete-all).
// Idempotent via the seam's 404-only delete. The find-and-delete-all path is what
// recovers a created-but-id-unresolved leaf — and it only works WHILE THE VM (the
// only listing that can find a nameless VMware leaf) still exists, which is why
// the caller runs this BEFORE deleting the VM.
func deleteVMwareLeaves(ctx context.Context, resolvedID string, findIDsByVM func() ([]string, error), deleteAndWait func(context.Context, string) error) error {
	if resolvedID != "" {
		return deleteAndWait(ctx, resolvedID)
	}
	ids, err := findIDsByVM()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if derr := deleteAndWait(ctx, id); derr != nil {
			return derr
		}
	}
	return nil
}

// vmwareDeprovision tears the assembled stack down leaves-first and, crucially,
// removes every VM-scoped leaf BEFORE deleting the VM anchor. This closes an
// orphan window the deferred teardown alone cannot: a created-but-id-unresolved
// VMware disk/adapter has NO caller-chosen name, so once the VM (the only listing
// that can find it) is gone it is unrecoverable. Sweeping while the VM still
// exists guarantees it is caught — never relying on the VM delete to cascade it.
//
// Every step is breaker-gated and recorded; deletes are idempotent (404-only) via
// the seams, so this safely overlaps the deferred TeardownAll (which remains the
// backstop when the breaker trips mid-cycle and the VM is therefore NOT deleted
// here, so its leaves are still listable). networkID == "" means no adapter was
// attached, so its delete step is a recorded skip.
func vmwareDeprovision(ctx context.Context, r *Run, cyc Cycle, vmID, networkID, adapterID, diskID string,
	adapterSeam vmwareAdapterSeam, diskSeam vmwareDiskSeam, vmSeam vmwareVMSeam) {

	// Track whether EVERY leaf removal was confirmed. A leaf op that ran and
	// returned an error (r.op returns non-nil only when fn actually ran and
	// failed; a breaker-skip returns nil) means a nameless leaf may still exist —
	// we must NOT then destroy the VM anchor, or that leaf becomes unrecoverable
	// (no name, and the VM-scoped listing that finds it is gone). Keep the VM so
	// the deferred TeardownAll can retry the sweep (and the VM) while it still
	// exists.
	leavesSwept := true

	if networkID != "" {
		if r.op(cyc, "compute.vmware.network_adapter.delete", func() error {
			return deleteVMwareLeaves(ctx, adapterID,
				func() ([]string, error) { return adapterSeam.FindIDsByVM(ctx, vmID) },
				adapterSeam.DeleteAndWait)
		}) != nil {
			leavesSwept = false
		}
	} else {
		r.skip(cyc, "compute.vmware.network_adapter.delete")
	}

	if r.op(cyc, "compute.vmware.virtual_disk.delete", func() error {
		return deleteVMwareLeaves(ctx, diskID,
			func() ([]string, error) { return diskSeam.FindIDsByVM(ctx, vmID) },
			diskSeam.DeleteAndWait)
	}) != nil {
		leavesSwept = false
	}

	// The VM anchor LAST — and ONLY if every leaf was confirmed gone. A failed leaf
	// removal leaves the VM in place (recorded skip) as the anchor for the deferred
	// retry; never destroy the anchor while a nameless leaf might still hang off it.
	if leavesSwept {
		_ = r.op(cyc, "compute.vmware.virtual_machine.delete", func() error {
			return vmSeam.DeleteAndWait(ctx, vmID)
		})
	} else {
		r.skip(cyc, "compute.vmware.virtual_machine.delete")
	}
}
