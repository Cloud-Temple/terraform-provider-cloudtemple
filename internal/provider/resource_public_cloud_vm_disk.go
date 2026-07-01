package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// vmDiskGrowOnlyCheck is the pure grow-only rule: a disk can only be extended,
// never shrunk. oldSize == 0 is the create case (no prior size) and is allowed.
func vmDiskGrowOnlyCheck(oldSize, newSize int) error {
	if oldSize != 0 && newSize < oldSize {
		return fmt.Errorf("size can only be increased (grow-only): %d GB -> %d GB is a shrink, which is not supported", oldSize, newSize)
	}
	return nil
}

const (
	vmDiskCreateTimeout = 45 * time.Minute
	vmDiskUpdateTimeout = 30 * time.Minute
	vmDiskDeleteTimeout = 30 * time.Minute
	vmDiskMaxSizeGb     = 2048
)

func resourcePublicCloudVMDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a DATA disk attached to a Public Cloud VM instance. The system disk is provided by the template and is not managed here. Create, extend (grow-only) and delete are asynchronous; extending and deleting a disk both require the VM to be stopped.",

		CreateContext: resourcePublicCloudVMDiskCreate,
		ReadContext:   resourcePublicCloudVMDiskRead,
		UpdateContext: resourcePublicCloudVMDiskUpdate,
		DeleteContext: resourcePublicCloudVMDiskDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importVMScopedResource(),
		},
		CustomizeDiff: customdiff.All(
			// Grow-only: a disk can only be extended, never shrunk.
			customdiff.ValidateChange("size", func(ctx context.Context, oldValue, newValue, meta any) error {
				return vmDiskGrowOnlyCheck(oldValue.(int), newValue.(int))
			}),
		),
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(vmDiskCreateTimeout),
			Update: schema.DefaultTimeout(vmDiskUpdateTimeout),
			Delete: schema.DefaultTimeout(vmDiskDeleteTimeout),
		},

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VM this data disk is attached to. Immutable.",
			},
			"size": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(1, vmDiskMaxSizeGb),
				Description:  "The size of the disk in GB (1-2048). Grow-only; changing it extends the disk, which requires the VM to be stopped.",
			},
			"storage_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the storage type. Immutable; when omitted the platform assigns a default.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The name (label) of the disk. Immutable; when omitted the platform assigns one.",
			},

			// Out
			"position": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The position of the disk on the VM (0 is the system disk; data disks are 1+).",
			},
			"is_primary": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is the primary (system) disk. Always false for a managed data disk.",
			},
		},
	}
}

// singleActivityResult returns the `result` of an activity's single terminal
// state, or "" if the activity is nil / has not exactly one state / has no
// result. It is NON-mutating (unlike setIdFromActivityState), so the caller can
// validate the candidate before adopting it as the resource id.
func singleActivityResult(a *client.Activity) string {
	if a == nil || len(a.State) != 1 {
		return ""
	}
	for _, s := range a.State {
		return s.Result
	}
	return ""
}

// vmDiskReadMode selects how a nil read is treated (see readVMDiskInto).
type vmDiskReadMode int

const (
	vmDiskReadForRefresh vmDiskReadMode = iota
	vmDiskReadAfterWrite
)

// vmDiskCRUDFuncs abstracts the client surface so the CRUD orchestration is
// unit-tested without HTTP. vmRead / vmListStrict resolve the PARENT VM for the
// parent-deleted absence rule and the extend "stopped" preflight.
type vmDiskCRUDFuncs struct {
	create       func(ctx context.Context, vmID string, req *client.CreateVMDiskRequest) (string, error)
	extend       func(ctx context.Context, vmID, diskID string, size int) (string, error)
	del          func(ctx context.Context, vmID, diskID string) (string, error)
	read         func(ctx context.Context, vmID, diskID string) (*client.PublicCloudVMDisk, error)
	listStrict   func(ctx context.Context, vmID string) ([]*client.PublicCloudVMDisk, error)
	vmRead       func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error)
	vmListStrict func(ctx context.Context) ([]*client.PublicCloudVMInstance, error)
	waitActivity func(ctx context.Context, activityID string) (*client.Activity, error)
}

func vmDiskClientFuncs(c *client.Client) vmDiskCRUDFuncs {
	disk := c.PublicCloudVM().Disk()
	inst := c.PublicCloudVM().Instance()
	return vmDiskCRUDFuncs{
		create:     disk.Create,
		extend:     disk.ExtendById,
		del:        disk.Delete,
		read:       disk.Read,
		listStrict: disk.ListStrict,
		vmRead:     inst.Read,
		vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
			return inst.ListStrict(ctx, &client.PublicCloudVMInstanceFilter{})
		},
		waitActivity: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, vmInstanceWaiterOptions(ctx))
		},
	}
}

func resourcePublicCloudVMDiskCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return createVMDiskWith(ctx, d, vmDiskClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readVMDiskInto(ctx, d, vmDiskClientFuncs(getClient(meta)), vmDiskReadForRefresh)
}

func resourcePublicCloudVMDiskUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return updateVMDiskWith(ctx, d, vmDiskClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMDiskDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return deleteVMDiskWith(ctx, d, vmDiskClientFuncs(getClient(meta)))
}

// createVMDiskWith: create a data disk, take the id ONLY from the completed
// activity's result (validated: non-empty, UUID, != vmID — never the vmId from
// concernedItems{vmi}), then read-back to assert it is a non-primary disk of the
// requested size.
func createVMDiskWith(ctx context.Context, d *schema.ResourceData, funcs vmDiskCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	req := &client.CreateVMDiskRequest{
		Size:        d.Get("size").(int),
		StorageType: d.Get("storage_type").(string),
		Name:        d.Get("name").(string),
	}

	activityID, err := funcs.create(ctx, vmID, req)
	if err != nil {
		return diag.Errorf("failed to create data disk on VM %s: %s", vmID, err)
	}
	activity, err := funcs.waitActivity(ctx, activityID)
	if err != nil {
		return diag.Errorf(
			"data disk create on VM %s: activity %q did not complete: %s. If a disk was created it is ORPHANED outside the state — audit the VM's disks and import it (terraform import <vmID>/<diskID>) or delete it before re-applying.",
			vmID, activityID, err,
		)
	}

	candidate := singleActivityResult(activity)
	if candidate == "" || !isUUID(candidate) || sameUUID(candidate, vmID) {
		return diag.Errorf(
			"data disk create on VM %s (activity %q) did not report a usable disk id (got %q); refusing to guess. Audit the VM's disks and import the new one if it was created.",
			vmID, activityID, candidate,
		)
	}
	d.SetId(candidate)

	// Validate the adopted id with a direct read BEFORE populating the state.
	// The distinction matters because SDKv2 persists d.State() even on a create
	// error: an UNREADABLE disk (error/nil) is eventual consistency — keep the id
	// so a later refresh reconciles it — while a read-back that PROVES a wrong
	// adoption (primary/system disk, or a size that is not the one we requested)
	// must clear the id: a tainted wrong id would later make Terraform DELETE the
	// wrong disk (data loss on the system disk).
	disk, rerr := funcs.read(ctx, vmID, candidate)
	if rerr != nil {
		return diag.Errorf("data disk %s of VM %s was just created but could not be read back: %s; the resource is kept in the state with its id.", candidate, vmID, rerr)
	}
	if disk == nil {
		return diag.Errorf("data disk %s of VM %s was just created but is not yet readable (eventual consistency); the resource is kept in the state with its id.", candidate, vmID)
	}
	if disk.IsPrimary {
		d.SetId("")
		return diag.Errorf("disk %s on VM %s read back as the primary (system) disk; a wrong id was adopted. The resource was NOT recorded in the state. If a data disk was created it is ORPHANED — audit the VM's disks and import (terraform import <vmID>/<diskID>) or delete it.", candidate, vmID)
	}
	if disk.SizeGb != req.Size {
		d.SetId("")
		return diag.Errorf("data disk %s on VM %s read back size %d GB, expected %d GB; a different disk was adopted. The resource was NOT recorded in the state. If a disk was created it is ORPHANED — audit the VM's disks and import (terraform import <vmID>/<diskID>) or delete it.", candidate, vmID, disk.SizeGb, req.Size)
	}
	return readVMDiskInto(ctx, d, funcs, vmDiskReadAfterWrite)
}

// readVMDiskInto holds the testable read logic. The resource is NEVER dropped on
// an inconclusive read (E0-9); a primary disk is never adopted.
func readVMDiskInto(ctx context.Context, d *schema.ResourceData, funcs vmDiskCRUDFuncs, mode vmDiskReadMode) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	diskID := d.Id()

	disk, err := funcs.read(ctx, vmID, diskID)
	if err != nil {
		return diag.Errorf("failed to read data disk %s of VM %s: %s. The resource is kept in the state (a forbidden or backend error is not proof of absence).", diskID, vmID, err)
	}
	if disk == nil {
		if mode == vmDiskReadAfterWrite {
			return diag.Errorf("data disk %s of VM %s was just written but is not yet readable (eventual consistency); the resource is kept in the state with its id.", diskID, vmID)
		}
		return dropVMDiskIfConfirmedAbsent(ctx, d, funcs, vmID, diskID)
	}

	// Never manage a primary/system disk: this resource is data-disk only.
	if disk.IsPrimary {
		return diag.Errorf("disk %s of VM %s is the primary (system) disk; this resource manages data disks only. Remove it from the Terraform state — it cannot be extended or deleted here.", diskID, vmID)
	}

	sw := newStateWriter(d)
	sw.set("virtual_machine_id", vmID)
	sw.set("size", disk.SizeGb)
	sw.set("storage_type", disk.StorageType)
	sw.set("name", disk.Label)
	sw.set("position", disk.Position)
	sw.set("is_primary", disk.IsPrimary)
	return sw.diags
}

// dropVMDiskIfConfirmedAbsent implements the E0-9 parent-deleted rule: drop the
// disk ONLY after authoritative absence — either the parent VM is proven gone
// (Read nil AND absent from a strict VM listing), or the disk is absent from a
// complete strict disk listing of an existing VM. Any ambiguity keeps the state.
func dropVMDiskIfConfirmedAbsent(ctx context.Context, d *schema.ResourceData, funcs vmDiskCRUDFuncs, vmID, diskID string) diag.Diagnostics {
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("data disk %s could not be read and the state of its VM %s could not be determined: %s; the resource is kept in the state.", diskID, vmID, err)
	}
	if vm == nil {
		// Parent VM appears gone — confirm with a strict VM listing before dropping.
		vms, lerr := funcs.vmListStrict(ctx)
		if lerr != nil {
			return diag.Errorf("data disk %s could not be read and the absence of its VM %s could not be confirmed (strict VM listing failed: %s); the resource is kept in the state.", diskID, vmID, lerr)
		}
		for _, listed := range vms {
			if listed != nil && sameUUID(listed.ID, vmID) {
				return diag.Errorf("data disk %s could not be read but its VM %s is still listed; refusing to drop it (possible access restriction).", diskID, vmID)
			}
		}
		// VM is authoritatively gone; a VM-scoped disk cannot outlive it.
		d.SetId("")
		return nil
	}

	// VM exists — confirm the disk's absence from a complete strict disk listing.
	disks, lerr := funcs.listStrict(ctx, vmID)
	if lerr != nil {
		return diag.Errorf("data disk %s could not be read and its deletion could not be confirmed (strict disk listing of VM %s failed: %s); the resource is kept in the state.", diskID, vmID, lerr)
	}
	for _, listed := range disks {
		if listed != nil && sameUUID(listed.ID, diskID) {
			return diag.Errorf("data disk %s could not be read but is still listed on VM %s; refusing to drop it (possible access restriction).", diskID, vmID)
		}
	}
	d.SetId("")
	return nil
}

// updateVMDiskWith: only `size` is mutable (extend). Preflight the VM state and
// refuse if it is not stopped or if the disk is primary (no auto-stop).
func updateVMDiskWith(ctx context.Context, d *schema.ResourceData, funcs vmDiskCRUDFuncs) diag.Diagnostics {
	if !d.HasChange("size") {
		return readVMDiskInto(ctx, d, funcs, vmDiskReadAfterWrite)
	}
	vmID := d.Get("virtual_machine_id").(string)
	diskID := d.Id()

	// Refuse to extend a primary disk (defence in depth; also guarded on delete).
	disk, err := funcs.read(ctx, vmID, diskID)
	if err != nil {
		return diag.Errorf("failed to read data disk %s of VM %s before extending: %s", diskID, vmID, err)
	}
	if disk == nil {
		return diag.Errorf("data disk %s of VM %s could not be found before extending; refusing to extend a disk that is not present.", diskID, vmID)
	}
	if disk.IsPrimary {
		return diag.Errorf("disk %s of VM %s is the primary (system) disk and cannot be extended by this resource.", diskID, vmID)
	}

	// Extend requires a stopped VM. Preflight and refuse clearly (no auto-stop).
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("failed to read VM %s before extending disk %s: %s", vmID, diskID, err)
	}
	if vm == nil {
		return diag.Errorf("VM %s could not be found before extending disk %s.", vmID, diskID)
	}
	if vm.Status != "stopped" {
		return diag.Errorf("disk %s cannot be extended while VM %s is %q: the VM must be stopped (set power_state = \"off\" on the VM, apply, then extend the disk).", diskID, vmID, vm.Status)
	}

	activityID, err := funcs.extend(ctx, vmID, diskID, d.Get("size").(int))
	if err != nil {
		return diag.Errorf("failed to extend data disk %s of VM %s: %s", diskID, vmID, err)
	}
	if _, err := funcs.waitActivity(ctx, activityID); err != nil {
		return diag.Errorf("data disk %s extend on VM %s: activity %q did not complete: %s", diskID, vmID, activityID, err)
	}
	return readVMDiskInto(ctx, d, funcs, vmDiskReadAfterWrite)
}

// deleteVMDiskWith: pre-read to refuse a primary disk, then delete async. A 404
// (disk or parent VM gone) is accepted only after a strict absence proof;
// 403/5xx/failed activity keep the state.
func deleteVMDiskWith(ctx context.Context, d *schema.ResourceData, funcs vmDiskCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	diskID := d.Id()

	disk, err := funcs.read(ctx, vmID, diskID)
	if err != nil {
		return diag.Errorf("failed to read data disk %s of VM %s before deleting: %s", diskID, vmID, err)
	}
	if disk == nil {
		// Already absent per a by-id read — confirm authoritatively before accepting.
		return dropVMDiskIfConfirmedAbsent(ctx, d, funcs, vmID, diskID)
	}
	if disk.IsPrimary {
		return diag.Errorf("disk %s of VM %s is the primary (system) disk and cannot be deleted.", diskID, vmID)
	}

	// Deleting a data disk requires the VM to be stopped (verified live: a delete
	// against a running VM fails with an opaque platform error and leaves the disk
	// in place). Preflight and refuse clearly (no hidden auto-stop), mirroring
	// extend.
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("failed to read VM %s before deleting disk %s: %s", vmID, diskID, err)
	}
	if vm == nil {
		return diag.Errorf("disk %s of VM %s is present but its VM could not be read; refusing to delete on an inconsistent/ambiguous signal.", diskID, vmID)
	}
	if vm.Status != "stopped" {
		return diag.Errorf("disk %s cannot be deleted while VM %s is %q: the VM must be stopped (set power_state = \"off\" on the VM, apply, then destroy the disk).", diskID, vmID, vm.Status)
	}

	activityID, err := funcs.del(ctx, vmID, diskID)
	if err != nil {
		// Race: the disk vanished between the pre-read and the delete. Accept the
		// destroy only after an authoritative absence proof (never on a bare 404).
		if isStatusCode(err, http.StatusNotFound) {
			return dropVMDiskIfConfirmedAbsent(ctx, d, funcs, vmID, diskID)
		}
		return diag.Errorf("failed to delete data disk %s of VM %s: %s", diskID, vmID, err)
	}
	if _, err := funcs.waitActivity(ctx, activityID); err != nil {
		return diag.Errorf("data disk %s delete on VM %s: activity %q did not complete: %s", diskID, vmID, activityID, err)
	}
	return nil
}
