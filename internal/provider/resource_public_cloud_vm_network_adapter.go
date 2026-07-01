package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	vmNICCreateTimeout = 45 * time.Minute
	vmNICUpdateTimeout = 30 * time.Minute
	vmNICDeleteTimeout = 30 * time.Minute
)

func resourcePublicCloudVMNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a network adapter (NIC) attached to a Public Cloud VM instance. Create, change-network and delete are asynchronous; changing the network and deleting the adapter both require the VM to be stopped. Attaching (create) is attempted hot but is not guaranteed on every network (hot-plug limitation).",

		CreateContext: resourcePublicCloudVMNetworkAdapterCreate,
		ReadContext:   resourcePublicCloudVMNetworkAdapterRead,
		UpdateContext: resourcePublicCloudVMNetworkAdapterUpdate,
		DeleteContext: resourcePublicCloudVMNetworkAdapterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importVMScopedResource(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(vmNICCreateTimeout),
			Update: schema.DefaultTimeout(vmNICUpdateTimeout),
			Delete: schema.DefaultTimeout(vmNICDeleteTimeout),
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VM this network adapter is attached to. Immutable.",
			},
			"device_index": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The device index of the adapter on the VM (0 = eth0, 1 = eth1, ...). Immutable; there is no API to change it.",
			},
			"network_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the network the adapter is attached to. Changing it re-points the adapter (change-network), which requires the VM to be stopped.",
			},
			"ip_address": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsIPv4Address,
				Description:  "A static IPv4 address to register for the adapter. Only honoured on VPC networks — it is silently ignored on Private Backbone networks. Write-only: it is never read back from the platform (the observed address is exposed as `ipv4_address`).",
			},

			// Out
			"network_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the attached network.",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The network type (`vpc` or `private_backbone`), computed by the platform from the attached network.",
			},
			"provision_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The provisioning status of the adapter (`provisioning`, `provisioned`, `failed`).",
			},
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The MAC address of the adapter. Populated after provisioning; may be empty until then.",
			},
			"ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The observed IPv4 address of the adapter. Reported by the guest agent after xe-guest-utilities is installed; may stay empty for a long time.",
			},
			"ipv6_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The observed IPv6 address of the adapter. Reported by the guest agent; may stay empty.",
			},
		},
	}
}

// vmNICReadMode selects how a nil/absent read is treated (see readVMNICInto).
type vmNICReadMode int

const (
	vmNICReadForRefresh vmNICReadMode = iota
	vmNICReadAfterWrite
)

// vmNICCRUDFuncs abstracts the client surface so the CRUD orchestration is
// unit-tested without HTTP. vmRead / vmListStrict resolve the PARENT VM for the
// parent-deleted absence rule and the "stopped" preflight of update/delete.
type vmNICCRUDFuncs struct {
	create        func(ctx context.Context, vmID string, req *client.CreateVMNetworkAdapterRequest) (string, error)
	changeNetwork func(ctx context.Context, vmID, nicID string, req *client.ChangeVMNetworkAdapterRequest) (string, error)
	del           func(ctx context.Context, vmID, nicID string) (string, error)
	read          func(ctx context.Context, vmID, nicID string) (*client.PublicCloudVMNetworkAdapter, error)
	listStrict    func(ctx context.Context, vmID string) ([]*client.PublicCloudVMNetworkAdapter, error)
	vmRead        func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error)
	vmListStrict  func(ctx context.Context) ([]*client.PublicCloudVMInstance, error)
	waitActivity  func(ctx context.Context, activityID string) (*client.Activity, error)
}

func vmNICClientFuncs(c *client.Client) vmNICCRUDFuncs {
	nic := c.PublicCloudVM().NetworkAdapter()
	inst := c.PublicCloudVM().Instance()
	return vmNICCRUDFuncs{
		create:        nic.Create,
		changeNetwork: nic.ChangeNetwork,
		del:           nic.Delete,
		read:          nic.Read,
		listStrict:    nic.ListStrict,
		vmRead:        inst.Read,
		vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
			return inst.ListStrict(ctx, &client.PublicCloudVMInstanceFilter{})
		},
		waitActivity: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, vmInstanceWaiterOptions(ctx))
		},
	}
}

func resourcePublicCloudVMNetworkAdapterCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return createVMNICWith(ctx, d, vmNICClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMNetworkAdapterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readVMNICInto(ctx, d, vmNICClientFuncs(getClient(meta)), vmNICReadForRefresh)
}

func resourcePublicCloudVMNetworkAdapterUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return updateVMNICWith(ctx, d, vmNICClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMNetworkAdapterDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return deleteVMNICWith(ctx, d, vmNICClientFuncs(getClient(meta)))
}

// createVMNICWith attaches a NIC, takes the id from the completed activity
// (concernedItem of type network_adapter first, else the single result — never
// the vmId), then reads back and asserts the adopted NIC's identity (id,
// device_index, network_id) matches what was requested. Any mismatch fails
// closed: we must not adopt a pre-existing NIC and orphan the one we created.
func createVMNICWith(ctx context.Context, d *schema.ResourceData, funcs vmNICCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	reqNetworkID := d.Get("network_id").(string)
	reqDeviceIndex := d.Get("device_index").(int)

	req := &client.CreateVMNetworkAdapterRequest{
		NetworkID:   reqNetworkID,
		DeviceIndex: reqDeviceIndex,
		IPAddress:   d.Get("ip_address").(string),
	}

	activityID, err := funcs.create(ctx, vmID, req)
	if err != nil {
		return diag.Errorf("failed to attach a network adapter to VM %s: %s", vmID, err)
	}
	activity, err := funcs.waitActivity(ctx, activityID)
	if err != nil {
		return diag.Errorf(
			"network adapter attach on VM %s: activity %q did not complete: %s. If an adapter was created it is ORPHANED outside the state — audit the VM's network adapters and import it (terraform import <vmID>/<networkAdapterID>) or delete it before re-applying.",
			vmID, activityID, err,
		)
	}

	// The new NIC id is the network_adapter concerned item; fall back to the single
	// activity result. Never the vmId.
	candidate := activityConcernedItemID(activity, "network_adapter")
	if candidate == "" {
		candidate = singleActivityResult(activity)
	}
	if candidate == "" || !isUUID(candidate) || sameUUID(candidate, vmID) {
		return diag.Errorf(
			"network adapter attach on VM %s (activity %q) did not report a usable adapter id (got %q); refusing to guess. Audit the VM's network adapters and import the new one if it was created.",
			vmID, activityID, candidate,
		)
	}
	d.SetId(candidate)

	diags := readVMNICInto(ctx, d, funcs, vmNICReadAfterWrite)
	if diags.HasError() {
		// The adapter was created (activity completed) but is not yet readable
		// (eventual consistency). KEEP the id so a later refresh reconciles it —
		// the id is the one we just created, not a wrong one.
		return diags
	}
	// Identity hardening: the read-back adapter must be the one we asked for. A
	// mismatch is PROVEN wrong-adoption (a different, pre-existing adapter), so we
	// must NOT leave that id in the state — SDKv2 persists d.State() even on an
	// error, and a tainted wrong id would later make Terraform DELETE the wrong
	// adapter. Clear the id and surface the orphan for manual cleanup.
	adoptedID := d.Id()
	if got := d.Get("device_index").(int); got != reqDeviceIndex {
		d.SetId("")
		return diag.Errorf("network adapter %s on VM %s read back device_index %d, expected %d; a different adapter was adopted. The resource was NOT recorded in the state. If an adapter was created it is ORPHANED — audit the VM's network adapters and import (terraform import <vmID>/<networkAdapterID>) or delete it.", adoptedID, vmID, got, reqDeviceIndex)
	}
	if got := d.Get("network_id").(string); !sameUUID(got, reqNetworkID) {
		d.SetId("")
		return diag.Errorf("network adapter %s on VM %s read back network_id %q, expected %q; a different adapter was adopted. The resource was NOT recorded in the state. If an adapter was created it is ORPHANED — audit the VM's network adapters and import or delete it.", adoptedID, vmID, got, reqNetworkID)
	}
	return diags
}

// readVMNICInto holds the testable read logic. The resource is NEVER dropped on
// an inconclusive read (E0-9): a by-id read failure (a 400 for an unknown NIC, a
// 403, a 5xx) is not proof of absence — only a complete listing that omits the
// NIC (or an authoritatively gone parent VM) drops it, and only on a refresh.
func readVMNICInto(ctx context.Context, d *schema.ResourceData, funcs vmNICCRUDFuncs, mode vmNICReadMode) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	nicID := d.Id()

	nic, err := funcs.read(ctx, vmID, nicID)
	if err != nil {
		if mode == vmNICReadAfterWrite {
			return diag.Errorf("network adapter %s of VM %s was just written but could not be read back: %s; the resource is kept in the state with its id.", nicID, vmID, err)
		}
		// Refresh: a by-id read error is ambiguous (the platform returns 400 for an
		// unknown NIC, not a clean 404). Never treat it as absence — decide via the
		// authoritative complete listings only.
		return dropVMNICIfConfirmedAbsent(ctx, d, funcs, vmID, nicID)
	}
	if nic == nil {
		if mode == vmNICReadAfterWrite {
			return diag.Errorf("network adapter %s of VM %s was just written but is not yet readable (eventual consistency); the resource is kept in the state with its id.", nicID, vmID)
		}
		return dropVMNICIfConfirmedAbsent(ctx, d, funcs, vmID, nicID)
	}
	// Defensive: a by-id read must return the adapter we asked for.
	if !sameUUID(nic.ID, nicID) {
		return diag.Errorf("network adapter read for %s of VM %s returned a mismatched id %q; refusing to refresh state on an inconsistent read.", nicID, vmID, nic.ID)
	}

	sw := newStateWriter(d)
	sw.set("virtual_machine_id", vmID)
	sw.set("device_index", nic.DeviceIndex)
	sw.set("network_id", nic.NetworkID)
	sw.set("network_name", nic.NetworkName)
	sw.set("type", nic.Type)
	sw.set("provision_status", nic.ProvisionStatus)
	sw.set("mac_address", nic.MacAddress)
	sw.set("ipv4_address", nic.IPv4Address)
	sw.set("ipv6_address", nic.IPv6Address)
	// ip_address is write-only: NEVER set it from the platform (the worker ignores
	// it on non-VPC networks, so reading it back from ipv4_address would create a
	// perpetual diff). Keep the configured value untouched.
	diags := sw.diags

	if ip := d.Get("ip_address").(string); ip != "" && nic.Type != "" && nic.Type != "vpc" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "ip_address has no effect on a non-VPC network",
			Detail:   fmt.Sprintf("network adapter %s is attached to a %q network; the static ip_address %q is only honoured on VPC networks and is ignored here.", nicID, nic.Type, ip),
		})
	}
	return diags
}

// dropVMNICIfConfirmedAbsent implements the E0-9 parent-deleted rule (an exact
// mirror of the disk resource): drop the NIC ONLY after authoritative absence —
// either the parent VM is proven gone (Read nil AND absent from a strict VM
// listing), or the NIC is absent from a complete strict listing of an existing
// VM. Any ambiguity (a read/list error, or the item still listed) keeps the state.
func dropVMNICIfConfirmedAbsent(ctx context.Context, d *schema.ResourceData, funcs vmNICCRUDFuncs, vmID, nicID string) diag.Diagnostics {
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("network adapter %s could not be read and the state of its VM %s could not be determined: %s; the resource is kept in the state.", nicID, vmID, err)
	}
	if vm == nil {
		// Parent VM appears gone — confirm with a strict VM listing before dropping.
		vms, lerr := funcs.vmListStrict(ctx)
		if lerr != nil {
			return diag.Errorf("network adapter %s could not be read and the absence of its VM %s could not be confirmed (strict VM listing failed: %s); the resource is kept in the state.", nicID, vmID, lerr)
		}
		for _, listed := range vms {
			if listed != nil && sameUUID(listed.ID, vmID) {
				return diag.Errorf("network adapter %s could not be read but its VM %s is still listed; refusing to drop it (possible access restriction).", nicID, vmID)
			}
		}
		// VM is authoritatively gone; a VM-scoped adapter cannot outlive it.
		d.SetId("")
		return nil
	}

	// VM exists — confirm the NIC's absence from a complete strict listing.
	nics, lerr := funcs.listStrict(ctx, vmID)
	if lerr != nil {
		return diag.Errorf("network adapter %s could not be read and its deletion could not be confirmed (strict network adapter listing of VM %s failed: %s); the resource is kept in the state.", nicID, vmID, lerr)
	}
	for _, listed := range nics {
		if listed != nil && sameUUID(listed.ID, nicID) {
			return diag.Errorf("network adapter %s could not be read but is still listed on VM %s; refusing to drop it (possible access restriction).", nicID, vmID)
		}
	}
	d.SetId("")
	return nil
}

// updateVMNICWith re-points the adapter to another network (change-network). Only
// network_id / ip_address are mutable; device_index and virtual_machine_id are
// ForceNew. The change requires a stopped VM — preflight and refuse clearly (no
// auto-stop), mirroring the disk resource. After the change, the read-back must
// confirm the network actually changed (the activity can complete without the
// hot-plug taking effect).
func updateVMNICWith(ctx context.Context, d *schema.ResourceData, funcs vmNICCRUDFuncs) diag.Diagnostics {
	if !d.HasChange("network_id") && !d.HasChange("ip_address") {
		return readVMNICInto(ctx, d, funcs, vmNICReadAfterWrite)
	}
	vmID := d.Get("virtual_machine_id").(string)
	nicID := d.Id()
	reqNetworkID := d.Get("network_id").(string)

	// Change-network requires a stopped VM. Preflight and refuse clearly.
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("failed to read VM %s before changing network adapter %s: %s", vmID, nicID, err)
	}
	if vm == nil {
		return diag.Errorf("VM %s could not be found before changing network adapter %s.", vmID, nicID)
	}
	if vm.Status != "stopped" {
		return diag.Errorf("network adapter %s cannot be changed while VM %s is %q: the VM must be stopped (set power_state = \"off\" on the VM, apply, then change the adapter).", nicID, vmID, vm.Status)
	}

	activityID, err := funcs.changeNetwork(ctx, vmID, nicID, &client.ChangeVMNetworkAdapterRequest{
		NetworkID: reqNetworkID,
		IPAddress: d.Get("ip_address").(string),
	})
	if err != nil {
		return diag.Errorf("failed to change the network of adapter %s on VM %s: %s", nicID, vmID, err)
	}
	if _, err := funcs.waitActivity(ctx, activityID); err != nil {
		return diag.Errorf("network adapter %s change-network on VM %s: activity %q did not complete: %s", nicID, vmID, activityID, err)
	}

	diags := readVMNICInto(ctx, d, funcs, vmNICReadAfterWrite)
	if diags.HasError() {
		return diags
	}
	// Confirm the change actually took effect. The activity can complete while the
	// hot-plug is a no-op on the hypervisor (network dette #46) — surface that
	// instead of silently reporting success.
	if got := d.Get("network_id").(string); !sameUUID(got, reqNetworkID) {
		return diag.Errorf("network adapter %s on VM %s still reports network_id %q after a change to %q; the change did not take effect (the network may not be hot-pluggable). The resource is kept in the state — verify the VM and network and re-apply.", nicID, vmID, got, reqNetworkID)
	}
	return diags
}

// deleteVMNICWith detaches the adapter. It pre-reads to accept an already-absent
// NIC only after an authoritative absence proof, then requires the VM to be
// stopped (worker guard) and deletes. A 404 race on delete is confirmed before
// being accepted; a 403/5xx/failed activity keeps the state.
func deleteVMNICWith(ctx context.Context, d *schema.ResourceData, funcs vmNICCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	nicID := d.Id()

	nic, err := funcs.read(ctx, vmID, nicID)
	if err != nil {
		// A by-id read error (400/403/5xx) is NOT proof of absence — confirm via the
		// authoritative listings before accepting the destroy.
		return dropVMNICIfConfirmedAbsent(ctx, d, funcs, vmID, nicID)
	}
	if nic == nil {
		// Already absent per a by-id read — confirm authoritatively before accepting.
		return dropVMNICIfConfirmedAbsent(ctx, d, funcs, vmID, nicID)
	}

	// Removing an adapter requires the VM to be stopped (worker guard: a delete
	// against a running VM fails and leaves the adapter in place). Preflight and
	// refuse clearly (no hidden auto-stop), mirroring the disk resource.
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("failed to read VM %s before deleting network adapter %s: %s", vmID, nicID, err)
	}
	if vm == nil {
		return diag.Errorf("network adapter %s of VM %s is present but its VM could not be read; refusing to delete on an inconsistent/ambiguous signal.", nicID, vmID)
	}
	if vm.Status != "stopped" {
		return diag.Errorf("network adapter %s cannot be removed while VM %s is %q: the VM must be stopped (set power_state = \"off\" on the VM, apply, then destroy the adapter).", nicID, vmID, vm.Status)
	}

	activityID, err := funcs.del(ctx, vmID, nicID)
	if err != nil {
		// Race: the adapter vanished between the pre-read and the delete. Accept the
		// destroy only after an authoritative absence proof (never on a bare 404).
		if isStatusCode(err, http.StatusNotFound) {
			return dropVMNICIfConfirmedAbsent(ctx, d, funcs, vmID, nicID)
		}
		return diag.Errorf("failed to delete network adapter %s of VM %s: %s", nicID, vmID, err)
	}
	if _, err := funcs.waitActivity(ctx, activityID); err != nil {
		return diag.Errorf("network adapter %s delete on VM %s: activity %q did not complete: %s", nicID, vmID, activityID, err)
	}
	return nil
}
