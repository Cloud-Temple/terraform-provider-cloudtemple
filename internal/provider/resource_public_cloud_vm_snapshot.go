package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	vmSnapshotCreateTimeout = 45 * time.Minute
	vmSnapshotDeleteTimeout = 30 * time.Minute
)

func resourcePublicCloudVMSnapshot() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a snapshot of a Public Cloud VM instance. Create and delete are asynchronous. The snapshot has no mutable attribute (name and virtual_machine_id are immutable), and reverting a VM to a snapshot is not exposed by this provider.",

		CreateContext: resourcePublicCloudVMSnapshotCreate,
		ReadContext:   resourcePublicCloudVMSnapshotRead,
		DeleteContext: resourcePublicCloudVMSnapshotDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importVMScopedResource(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(vmSnapshotCreateTimeout),
			Delete: schema.DefaultTimeout(vmSnapshotDeleteTimeout),
		},

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VM to snapshot. Immutable.",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
				Description:  "The name of the snapshot. Immutable (renaming recreates the snapshot).",
			},

			// Out
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the snapshot (e.g. `available`).",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the snapshot (RFC3339).",
			},
		},
	}
}

// vmSnapshotReadMode selects how a nil read is treated (see readVMSnapshotInto).
type vmSnapshotReadMode int

const (
	vmSnapshotReadForRefresh vmSnapshotReadMode = iota
	vmSnapshotReadAfterWrite
)

// vmSnapshotCRUDFuncs abstracts the client surface for unit testing without HTTP.
type vmSnapshotCRUDFuncs struct {
	create       func(ctx context.Context, vmID, name string) (string, error)
	del          func(ctx context.Context, vmID, snapshotID string) (string, error)
	read         func(ctx context.Context, vmID, snapshotID string) (*client.PublicCloudVMSnapshot, error)
	listStrict   func(ctx context.Context, vmID string) ([]*client.PublicCloudVMSnapshot, error)
	vmRead       func(ctx context.Context, vmID string) (*client.PublicCloudVMInstance, error)
	vmListStrict func(ctx context.Context) ([]*client.PublicCloudVMInstance, error)
	waitActivity func(ctx context.Context, activityID string) (*client.Activity, error)
}

func vmSnapshotClientFuncs(c *client.Client) vmSnapshotCRUDFuncs {
	snap := c.PublicCloudVM().Snapshot()
	inst := c.PublicCloudVM().Instance()
	return vmSnapshotCRUDFuncs{
		create:     snap.Create,
		del:        snap.Delete,
		read:       snap.Read,
		listStrict: snap.ListStrict,
		vmRead:     inst.Read,
		vmListStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
			return inst.ListStrict(ctx, &client.PublicCloudVMInstanceFilter{})
		},
		waitActivity: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, vmInstanceWaiterOptions(ctx))
		},
	}
}

func resourcePublicCloudVMSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return createVMSnapshotWith(ctx, d, vmSnapshotClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMSnapshotRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readVMSnapshotInto(ctx, d, vmSnapshotClientFuncs(getClient(meta)), vmSnapshotReadForRefresh)
}

func resourcePublicCloudVMSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Get("virtual_machine_id").(string))
	defer unlock()
	return deleteVMSnapshotWith(ctx, d, vmSnapshotClientFuncs(getClient(meta)))
}

// createVMSnapshotWith: the snapshot id is taken from the completed activity —
// the concernedItems "snapshot" entry (present live), else the terminal state
// result — validated (non-empty, UUID, != vmID) before SetId.
func createVMSnapshotWith(ctx context.Context, d *schema.ResourceData, funcs vmSnapshotCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	name := d.Get("name").(string)

	activityID, err := funcs.create(ctx, vmID, name)
	if err != nil {
		return diag.Errorf("failed to create snapshot %q of VM %s: %s", name, vmID, err)
	}
	activity, err := funcs.waitActivity(ctx, activityID)
	if err != nil {
		return diag.Errorf(
			"snapshot %q create on VM %s: activity %q did not complete: %s. If a snapshot was created it is ORPHANED outside the state — audit the VM's snapshots and import it (terraform import <vmID>/<snapshotID>) or delete it before re-applying.",
			name, vmID, activityID, err,
		)
	}

	candidate := activityConcernedItemID(activity, "snapshot")
	if candidate == "" {
		candidate = singleActivityResult(activity)
	}
	if candidate == "" || !isUUID(candidate) || sameUUID(candidate, vmID) {
		return diag.Errorf(
			"snapshot %q create on VM %s (activity %q) did not report a usable snapshot id (got %q); refusing to guess. Audit the VM's snapshots and import the new one if it was created.",
			name, vmID, activityID, candidate,
		)
	}
	d.SetId(candidate)

	return readVMSnapshotInto(ctx, d, funcs, vmSnapshotReadAfterWrite)
}

// readVMSnapshotInto: never dropped on an inconclusive read (E0-9). A nil refresh
// drops only after authoritative absence — the parent VM proven gone, or the
// snapshot absent from a complete strict snapshot listing of an existing VM.
func readVMSnapshotInto(ctx context.Context, d *schema.ResourceData, funcs vmSnapshotCRUDFuncs, mode vmSnapshotReadMode) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	snapshotID := d.Id()

	snap, err := funcs.read(ctx, vmID, snapshotID)
	if err != nil {
		return diag.Errorf("failed to read snapshot %s of VM %s: %s. The resource is kept in the state (a forbidden or backend error is not proof of absence).", snapshotID, vmID, err)
	}
	if snap == nil {
		if mode == vmSnapshotReadAfterWrite {
			return diag.Errorf("snapshot %s of VM %s was just written but is not yet readable (eventual consistency); the resource is kept in the state with its id.", snapshotID, vmID)
		}
		return dropVMSnapshotIfConfirmedAbsent(ctx, d, funcs, vmID, snapshotID)
	}

	sw := newStateWriter(d)
	sw.set("virtual_machine_id", vmID)
	sw.set("name", snap.Name)
	sw.set("status", snap.Status)
	sw.set("created_at", snap.CreatedAt)
	return sw.diags
}

// dropVMSnapshotIfConfirmedAbsent implements the E0-9 parent-deleted rule (see the
// disk equivalent): drop ONLY after authoritative absence; any ambiguity keeps.
func dropVMSnapshotIfConfirmedAbsent(ctx context.Context, d *schema.ResourceData, funcs vmSnapshotCRUDFuncs, vmID, snapshotID string) diag.Diagnostics {
	vm, err := funcs.vmRead(ctx, vmID)
	if err != nil {
		return diag.Errorf("snapshot %s could not be read and the state of its VM %s could not be determined: %s; the resource is kept in the state.", snapshotID, vmID, err)
	}
	if vm == nil {
		vms, lerr := funcs.vmListStrict(ctx)
		if lerr != nil {
			return diag.Errorf("snapshot %s could not be read and the absence of its VM %s could not be confirmed (strict VM listing failed: %s); the resource is kept in the state.", snapshotID, vmID, lerr)
		}
		for _, listed := range vms {
			if listed != nil && sameUUID(listed.ID, vmID) {
				return diag.Errorf("snapshot %s could not be read but its VM %s is still listed; refusing to drop it (possible access restriction).", snapshotID, vmID)
			}
		}
		d.SetId("")
		return nil
	}

	snaps, lerr := funcs.listStrict(ctx, vmID)
	if lerr != nil {
		return diag.Errorf("snapshot %s could not be read and its deletion could not be confirmed (strict snapshot listing of VM %s failed: %s); the resource is kept in the state.", snapshotID, vmID, lerr)
	}
	for _, listed := range snaps {
		if listed != nil && sameUUID(listed.ID, snapshotID) {
			return diag.Errorf("snapshot %s could not be read but is still listed on VM %s; refusing to drop it (possible access restriction).", snapshotID, vmID)
		}
	}
	d.SetId("")
	return nil
}

// deleteVMSnapshotWith: delete async; a 404 (snapshot or parent VM gone) is
// accepted only after a strict absence proof; 403/5xx/failed activity keep state.
func deleteVMSnapshotWith(ctx context.Context, d *schema.ResourceData, funcs vmSnapshotCRUDFuncs) diag.Diagnostics {
	vmID := d.Get("virtual_machine_id").(string)
	snapshotID := d.Id()

	activityID, err := funcs.del(ctx, vmID, snapshotID)
	if err != nil {
		if isStatusCode(err, http.StatusNotFound) {
			return dropVMSnapshotIfConfirmedAbsent(ctx, d, funcs, vmID, snapshotID)
		}
		return diag.Errorf("failed to delete snapshot %s of VM %s: %s", snapshotID, vmID, err)
	}
	if _, err := funcs.waitActivity(ctx, activityID); err != nil {
		return diag.Errorf("snapshot %s delete on VM %s: activity %q did not complete: %s", snapshotID, vmID, activityID, err)
	}
	return nil
}
