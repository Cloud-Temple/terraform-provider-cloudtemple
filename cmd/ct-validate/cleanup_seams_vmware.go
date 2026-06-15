package main

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// VMware (vCenter) compute lifecycle teardowns — the sibling of the OpenIaaS
// teardowns in cleanup_seams.go, with the SAME doctrine:
//   - registered BEFORE the create they undo (F3), keyed by a deterministic
//     run-unique identity, so a created-but-unresolved resource is still swept;
//   - leaves-first under LIFO (adapter, then data disk, then the VM anchor): a VM
//     delete must not be relied on to cascade a user disk/adapter;
//   - find-by-name via a STRICT listing when the create activity did not resolve
//     the id, FAIL CLOSED on >1 match (the run-unique name matches at most one);
//   - 404-only idempotent deletes (shared idempotentDeleteErr): only a definitive
//     not-found proves absence — a 403/409 is NOT "already gone" (mirrors the
//     #303/#325 doctrine), so a compute DELETE that 403s for an absent resource
//     surfaces here as a teardown FAILURE (a false alarm, never an orphan).
//
// Each registration takes a narrow SEAM interface (only the methods it needs),
// not *client.Client, so it is unit-testable offline with a fake that returns an
// error AFTER simulating the server-side effect.

// --- VMware virtual machine ---------------------------------------------------

// vmwareVMSeam is the subset of the VMware VM client a VM teardown needs.
type vmwareVMSeam interface {
	DeleteAndWait(ctx context.Context, id string) error
	PowerOffAndWait(ctx context.Context, id string) error // best-effort, never fatal
	FindIDByName(ctx context.Context, name, datacenterID string) (string, error)
}

// vmwareVMTeardownRef carries the VM identity; ID is filled once the create
// activity resolves it (shared pointer). DatacenterID scopes the fallback
// find-by-name to a bounded, strict listing.
type vmwareVMTeardownRef struct {
	Name         string
	DatacenterID string
	ID           string
	Resolved     bool
}

// registerVMwareVMTeardown registers the VM teardown (the anchor; runs LAST under
// LIFO). Best-effort power-off (a running VM can refuse delete) then delete; if
// the id never resolved, find the VM by its deterministic name within the
// datacenter and delete that.
func registerVMwareVMTeardown(cl *Cleanup, seam vmwareVMSeam, ref *vmwareVMTeardownRef) {
	cl.Register(fmt.Sprintf("compute.vmware.virtual_machine %s", ref.Name), func(tctx context.Context) error {
		id := ref.ID
		if !ref.Resolved || id == "" {
			found, err := seam.FindIDByName(tctx, ref.Name, ref.DatacenterID)
			if err != nil {
				return err
			}
			if found == "" {
				return nil // never created / already gone → idempotent success
			}
			id = found
		}
		_ = seam.PowerOffAndWait(tctx, id) // best-effort: a powered-on VM may refuse delete
		return seam.DeleteAndWait(tctx, id)
	})
}

type computeVMwareVMSeam struct{ c *client.Client }

func (s computeVMwareVMSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().VirtualMachine().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err) // 404 → already gone; other surfaced
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr // a failed delete activity is a real teardown failure
}

func (s computeVMwareVMSeam) PowerOffAndWait(ctx context.Context, id string) error {
	vm, err := s.c.Compute().VirtualMachine().Read(ctx, id)
	if err != nil || vm == nil {
		return nil // best-effort: absent or unreadable → the subsequent delete decides
	}
	if vm.PowerState != "running" {
		return nil // already off → nothing to do
	}
	activityID, perr := s.c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
		ID:           id,
		DatacenterId: vm.Datacenter.ID,
		PowerAction:  "off",
	})
	if perr != nil {
		return nil // best-effort; the subsequent delete surfaces a real problem
	}
	_, _ = s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return nil
}

func (s computeVMwareVMSeam) FindIDByName(ctx context.Context, name, datacenterID string) (string, error) {
	vms, err := s.c.Compute().VirtualMachine().ListStrict(ctx,
		&client.VirtualMachineFilter{Datacenters: []string{datacenterID}})
	if err != nil {
		return "", err
	}
	var found string
	for _, vm := range vms {
		if vm != nil && vm.Name == name && vm.ID != "" {
			if found != "" {
				// Ambiguous: the run-unique name should match at most one. More than
				// one means an anomaly — fail closed (surface), never delete a
				// possibly-wrong VM.
				return "", fmt.Errorf("ambiguous: more than one VMware virtual machine named %q", name)
			}
			found = vm.ID
		}
	}
	return found, nil
}

// --- VMware virtual disk ------------------------------------------------------

// vmwareDiskSeam is the subset of the VMware virtual-disk client a disk teardown
// needs. CRITICAL: CreateVirtualDiskRequest carries NO name (unlike OpenIaaS) and
// REQUIRES a virtualMachineId — a VMware disk is always created ATTACHED to a VM
// and gets a platform-assigned name. So a created-but-id-unresolved disk cannot
// be found by a name we chose; it is recovered by listing the VM's disks and
// deleting them. The VM is run-unique and ours, and the cycle creates it via the
// bare Create (no template/clone/library) — which yields a VM with no disks until
// this cycle attaches one — so every disk subsequently on it was attached by this
// cycle. Hence the delete-all-on-VM strategy (same as the adapter) cannot remove a
// foreign disk. There is no unattached-disk orphan window (the API cannot create a
// VMware disk without a VM), so a VM-scoped listing is complete.
type vmwareDiskSeam interface {
	FindIDsByVM(ctx context.Context, vmID string) ([]string, error)
	DeleteAndWait(ctx context.Context, id string) error
}

type vmwareDiskTeardownRef struct {
	VMID     string
	ID       string
	Resolved bool
}

// registerVMwareDiskTeardown registers the user data-disk teardown (a leaf; runs
// before the VM teardown under LIFO — a VM delete must not be relied on to remove
// a user disk cleanly). When the id resolved, delete it; otherwise delete every
// disk on the (run-unique, ours, created-diskless) VM. Registered before the disk
// create (F3).
func registerVMwareDiskTeardown(cl *Cleanup, seam vmwareDiskSeam, ref *vmwareDiskTeardownRef) {
	cl.Register(fmt.Sprintf("compute.vmware.virtual_disk on %s", ref.VMID), func(tctx context.Context) error {
		if ref.Resolved && ref.ID != "" {
			return seam.DeleteAndWait(tctx, ref.ID)
		}
		ids, err := seam.FindIDsByVM(tctx, ref.VMID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if derr := seam.DeleteAndWait(tctx, id); derr != nil {
				return derr
			}
		}
		return nil
	})
}

type computeVMwareDiskSeam struct{ c *client.Client }

func (s computeVMwareDiskSeam) FindIDsByVM(ctx context.Context, vmID string) ([]string, error) {
	disks, err := s.c.Compute().VirtualDisk().ListStrict(ctx, &client.VirtualDiskFilter{VirtualMachineID: vmID})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, d := range disks {
		if d != nil && d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

func (s computeVMwareDiskSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().VirtualDisk().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err)
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}

// --- VMware network adapter ---------------------------------------------------

// vmwareAdapterSeam is the subset of the VMware network-adapter client an adapter
// teardown needs.
type vmwareAdapterSeam interface {
	// FindIDsByVM returns the ids of EVERY adapter on the VM. For a
	// created-but-unresolved adapter the teardown deletes all of them: the VM is
	// run-unique and ours and is being destroyed, so removing its adapters is safe,
	// and there is no MAC to match on (the platform assigns it).
	FindIDsByVM(ctx context.Context, vmID string) ([]string, error)
	DeleteAndWait(ctx context.Context, id string) error
}

type vmwareAdapterTeardownRef struct {
	VMID     string
	ID       string
	Resolved bool
}

// registerVMwareNetworkAdapterTeardown registers the adapter teardown (a network
// leaf; runs FIRST under LIFO). When the id resolved, delete it; otherwise delete
// every adapter on the (run-unique, ours) VM.
func registerVMwareNetworkAdapterTeardown(cl *Cleanup, seam vmwareAdapterSeam, ref *vmwareAdapterTeardownRef) {
	cl.Register(fmt.Sprintf("compute.vmware.network_adapter %s", ref.VMID), func(tctx context.Context) error {
		if ref.Resolved && ref.ID != "" {
			return seam.DeleteAndWait(tctx, ref.ID)
		}
		ids, err := seam.FindIDsByVM(tctx, ref.VMID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if derr := seam.DeleteAndWait(tctx, id); derr != nil {
				return derr
			}
		}
		return nil
	})
}

type computeVMwareAdapterSeam struct{ c *client.Client }

func (s computeVMwareAdapterSeam) FindIDsByVM(ctx context.Context, vmID string) ([]string, error) {
	adapters, err := s.c.Compute().NetworkAdapter().ListStrict(ctx,
		&client.NetworkAdapterFilter{VirtualMachineID: vmID})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, a := range adapters {
		if a != nil && a.ID != "" {
			ids = append(ids, a.ID)
		}
	}
	return ids, nil
}

func (s computeVMwareAdapterSeam) DeleteAndWait(ctx context.Context, id string) error {
	activityID, err := s.c.Compute().NetworkAdapter().Delete(ctx, id)
	if err != nil {
		return idempotentDeleteErr(err)
	}
	_, werr := s.c.Activity().WaitForCompletion(ctx, activityID, silentWaiter)
	return werr
}
