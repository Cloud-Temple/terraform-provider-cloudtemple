package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// vmInstanceActivityNotFoundRetries widens the activity waiter's initial
// not-found tolerance (E0-7) for VM writes: right after a write returns its
// Location, the activity may take a few reads to become indexed. A budget of 5
// prevents failing (and orphaning) a write that is still running platform-side.
const vmInstanceActivityNotFoundRetries = 5

// Timeout budget (E0-4). The upstream worker monitors a VM task for up to 30
// minutes (VMI_TASK_MONITORING_TIMEOUT); create is the longest (provision +
// boot), so it gets the widest budget. The waiter honours the ctx deadline these
// blocks install.
const (
	vmInstanceCreateTimeout = 45 * time.Minute
	vmInstanceUpdateTimeout = 30 * time.Minute
	vmInstanceDeleteTimeout = 30 * time.Minute
)

func resourcePublicCloudVMInstance() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Public Cloud VM instance. Creation, resize, power transitions and deletion are asynchronous (tracked through the Shiva Activities service). The system disk is provided by the template and is not created here; data disks and additional network adapters are managed by their own resources.",

		CreateContext: resourcePublicCloudVMInstanceCreate,
		ReadContext:   resourcePublicCloudVMInstanceRead,
		UpdateContext: resourcePublicCloudVMInstanceUpdate,
		DeleteContext: resourcePublicCloudVMInstanceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: customizeVMInstanceDiff,

		// Generous timeouts: every write is async and a created VM must never be
		// abandoned mid-flight by a premature timeout (the id is set from the
		// completed create activity, so a later timeout cannot orphan it).
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(vmInstanceCreateTimeout),
			Update: schema.DefaultTimeout(vmInstanceUpdateTimeout),
			Delete: schema.DefaultTimeout(vmInstanceDeleteTimeout),
		},

		Schema: map[string]*schema.Schema{
			// In — mutable in place
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of the virtual machine. Mutable (issues a metadata update).",
			},
			"cpu": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The number of vCPUs. Mutable via resize, which requires `power_state = \"off\"`.",
			},
			"memory": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "The amount of RAM in GB. Mutable via resize, which requires `power_state = \"off\"`.",
			},
			"backup_policy_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the backup policy applied to the VM. Required, mutable (issues a metadata update).",
			},
			"power_state": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "off",
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
				Description:  "The desired power state (`on` or `off`, default `off`). Honoured from the first apply (passed to the create call, so an `on` VM boots at creation). Changing it later issues a start (`off`->`on`) or stop (`on`->`off`).",
			},

			// In — immutable (ForceNew): no endpoint updates them in place
			"availability_zone_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the availability zone where the VM is placed. Immutable.",
			},
			"template_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the OS template the VM is created from. Immutable.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the instance family. Immutable.",
			},
			"os_network_adapter": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MinItems:    1,
				MaxItems:    8,
				Description: "The network interfaces attached at creation (Private Backbone networks only — attach VPC networks with the dedicated network adapter resource). Immutable here; additional adapters are managed by the dedicated network adapter resource.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_index": {
							Type:         schema.TypeInt,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntAtLeast(0),
							Description:  "The device index (order) of the interface on the VM.",
						},
						"network_id": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IsUUID,
							Description:  "The ID of the network the interface is attached to.",
						},
						"ip_address": {
							Type:         schema.TypeString,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.IsIPv4Address,
							Description:  "The fixed IPv4 address to assign. When omitted, the platform assigns one.",
						},
					},
				},
			},
			"cloud_init": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "The cloud-init configuration applied at creation (keys `cloud_config` and/or `network_config`). Immutable and not readable back, so it is not reconciled on refresh.",
				Elem:        &schema.Schema{Type: schema.TypeString},
				ValidateDiagFunc: validation.MapKeyMatch(
					regexp.MustCompile("^cloud_config$|^network_config$"),
					"the only allowed cloud_init keys are cloud_config and network_config",
				),
			},

			// Out
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current status of the VM (e.g. `running`, `stopped`).",
			},
			"disks_size_gb": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The total size of the VM's disks (system + data) in GB.",
			},
			"guest_tools_installed": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the guest tools are installed.",
			},
			"availability_zone_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the availability zone.",
			},
			"template_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the OS template.",
			},
			"instance_family_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the instance family.",
			},
			"backup_policy_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the applied backup policy.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the VM (RFC3339).",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last update date of the VM (RFC3339).",
			},
			"os_disk": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: "The system (primary) disk of the VM, provided by the template. Declare the block with `size_gb` to grow it (grow-only; requires the VM to be stopped). Not settable at creation — the template's size is used. Data disks are managed by the separate disk resource.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the system disk."},
						"size_gb": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntAtLeast(1),
							Description:  "The size of the system disk in GB. Grow-only; increasing it extends the system disk, which requires the VM to be stopped. When omitted, the current size is kept.",
						},
						"storage_type": {Type: schema.TypeString, Computed: true, Description: "The ID of the storage type."},
						"position":     {Type: schema.TypeInt, Computed: true, Description: "The position of the disk (0 for the system disk)."},
						"is_primary":   {Type: schema.TypeBool, Computed: true, Description: "Always true for the system disk."},
					},
				},
			},
		},
	}
}

// customizeVMInstanceDiff enforces the resize precondition at PLAN time: cpu and
// memory can only change while the VM is declared stopped. This is checked ONLY
// on an existing resource — on create, cpu/memory necessarily "change" from their
// zero value and an `on` power_state is the legitimate boot-at-create case.
func customizeVMInstanceDiff(ctx context.Context, d *schema.ResourceDiff, meta any) error {
	exists := d.Id() != ""
	// os_disk.size_gb cannot be set at CREATE: the VM is created with the
	// template's system disk size, and create never extends it — so a configured
	// value that differs from the template would produce an inconsistent result.
	// Detected on the RAW config (never the diff/state: the attribute is
	// Optional+Computed, so d.Get would report a computed value as "set").
	if !exists {
		if osDiskSizeSetInRawConfig(d.GetRawConfig()) {
			return fmt.Errorf("os_disk.size_gb cannot be set when creating the VM: the template's system disk size is used at creation. Omit it, then set it in a later apply (with power_state = \"off\") to grow the system disk")
		}
	} else if d.HasChange("os_disk.0.size_gb") {
		// Grow-only + requires-off, validated on the LEAF only: a refresh of a
		// Computed sibling (id, storage_type, ...) must never look like an extend.
		o, n := d.GetChange("os_disk.0.size_gb")
		oldSize, okOld := o.(int)
		newSize, okNew := n.(int)
		if okOld && okNew {
			if err := vmInstanceOSDiskChangeCheck(oldSize, newSize, d.Get("power_state").(string)); err != nil {
				return err
			}
		}
	}
	return vmInstanceResizeRequiresOff(exists, d.HasChange("cpu") || d.HasChange("memory"), d.Get("power_state").(string))
}

// osDiskSizeSetInRawConfig reports whether the raw config declares an os_disk
// block with an explicitly set size_gb (null-, absent- and unknown-safe: an
// unknown value counts as set, since create cannot honour it either). It reads
// the RAW config because os_disk.size_gb is Optional+Computed — the diff/state
// view cannot tell an explicit value from a computed one.
func osDiskSizeSetInRawConfig(raw cty.Value) bool {
	if raw.IsNull() || !raw.Type().IsObjectType() || !raw.Type().HasAttribute("os_disk") {
		return false
	}
	osd := raw.GetAttr("os_disk")
	if osd.IsNull() {
		return false
	}
	// A declared-but-unknown os_disk cannot be inspected — and create cannot
	// honour a size that only resolves later. Reject conservatively.
	if !osd.IsKnown() {
		return true
	}
	if !osd.CanIterateElements() {
		return false
	}
	for it := osd.ElementIterator(); it.Next(); {
		_, el := it.Element()
		if el.IsNull() || !el.Type().IsObjectType() || !el.Type().HasAttribute("size_gb") {
			continue
		}
		if sg := el.GetAttr("size_gb"); !sg.IsNull() {
			return true
		}
	}
	return false
}

// vmInstanceOSDiskChangeCheck is the pure os_disk.size_gb change rule on an
// EXISTING VM. A zero new size (block removed, value resolved by Computed) is
// not a change to validate. A zero OLD size (no readable primary in state)
// cannot prove a shrink — the grow-only check is skipped — but the update will
// still issue an extend, so the stopped-VM precondition applies regardless.
func vmInstanceOSDiskChangeCheck(oldSize, newSize int, powerState string) error {
	if newSize == 0 || newSize == oldSize {
		return nil
	}
	if oldSize != 0 {
		if err := vmDiskGrowOnlyCheck(oldSize, newSize); err != nil {
			return err
		}
	}
	if powerState != "off" {
		return fmt.Errorf("os_disk.size_gb can only be changed while power_state = \"off\" (extending the system disk requires a stopped VM); set power_state = \"off\" in the same change, then power the VM back on in a subsequent apply")
	}
	return nil
}

// vmInstanceResizeRequiresOff is the pure resize precondition: on an EXISTING VM,
// cpu/memory can only change while power_state is declared "off". On create
// (exists == false) it never fires — cpu/memory necessarily change from zero and
// an "on" power_state is the legitimate boot-at-create case.
func vmInstanceResizeRequiresOff(exists, resizing bool, powerState string) error {
	if exists && resizing && powerState != "off" {
		return fmt.Errorf("cpu/memory can only be changed while power_state = \"off\" (a resize requires a stopped VM); set power_state = \"off\" in the same change to resize, then power the VM back on in a subsequent apply")
	}
	return nil
}

// vmInstanceReadMode selects how a nil (404) read is treated.
type vmInstanceReadMode int

const (
	// vmInstanceReadForRefresh: a confirmed absence is genuine deletion evidence
	// (after a strict-listing confirmation) -> drop the resource.
	vmInstanceReadForRefresh vmInstanceReadMode = iota
	// vmInstanceReadAfterWrite: right after a write the VM's existence is positive
	// evidence, so a nil read is eventual consistency, NOT a deletion -> fail closed
	// keeping the id (never orphan a just-written VM).
	vmInstanceReadAfterWrite
)

// vmInstanceCRUDFuncs abstracts the client surface so the CRUD orchestration is
// unit-tested with injected fakes, without HTTP calls or sleeps. The write funcs
// return the raw activityId; waitActivity blocks until the activity is terminal
// and returns it (so the create can extract the VM id from it).
type vmInstanceCRUDFuncs struct {
	create       func(ctx context.Context, req *client.CreateVMInstanceRequest) (string, error)
	patch        func(ctx context.Context, id string, req *client.PatchVMInstanceRequest) (string, error)
	resize       func(ctx context.Context, id string, req *client.ResizeVMInstanceRequest) (string, error)
	start        func(ctx context.Context, id string) (string, error)
	stop         func(ctx context.Context, id string) (string, error)
	del          func(ctx context.Context, id string) (string, error)
	read         func(ctx context.Context, id string) (*client.PublicCloudVMInstance, error)
	listStrict   func(ctx context.Context) ([]*client.PublicCloudVMInstance, error)
	listDisks    func(ctx context.Context, id string) ([]*client.PublicCloudVMDisk, error)
	extendSystem func(ctx context.Context, id string, size int) (string, error)
	networkRead  func(ctx context.Context, id string) (*client.PublicCloudVMNetwork, error)
	waitActivity func(ctx context.Context, activityID string) (*client.Activity, error)
}

func vmInstanceWaiterOptions(ctx context.Context) *client.WaiterOptions {
	o := getWaiterOptions(ctx)
	o.NotFoundRetries = vmInstanceActivityNotFoundRetries
	return o
}

func vmInstanceClientFuncs(c *client.Client) vmInstanceCRUDFuncs {
	inst := c.PublicCloudVM().Instance()
	return vmInstanceCRUDFuncs{
		create: inst.Create,
		patch:  inst.PatchMetadata,
		resize: inst.Resize,
		start:  inst.Start,
		stop:   inst.Stop,
		del:    inst.Delete,
		read:   inst.Read,
		listStrict: func(ctx context.Context) ([]*client.PublicCloudVMInstance, error) {
			return inst.ListStrict(ctx, &client.PublicCloudVMInstanceFilter{})
		},
		listDisks:    c.PublicCloudVM().Disk().List,
		extendSystem: c.PublicCloudVM().Disk().ExtendSystem,
		networkRead:  c.PublicCloudVM().Network().Read,
		waitActivity: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, vmInstanceWaiterOptions(ctx))
		},
	}
}

func resourcePublicCloudVMInstanceCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// No per-VM mutex at create: there is no VM id yet and each create is a
	// distinct VM (nothing to serialize against).
	return createVMInstanceWith(ctx, d, vmInstanceClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMInstanceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return readVMInstanceInto(ctx, d, vmInstanceClientFuncs(getClient(meta)), vmInstanceReadForRefresh)
}

func resourcePublicCloudVMInstanceUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Id())
	defer unlock()
	return updateVMInstanceWith(ctx, d, vmInstanceClientFuncs(getClient(meta)))
}

func resourcePublicCloudVMInstanceDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	unlock := publicCloudVMInstanceMutex.lock(d.Id())
	defer unlock()
	return deleteVMInstanceWith(ctx, d, vmInstanceClientFuncs(getClient(meta)))
}

// createVMInstanceWith holds the testable create orchestration. State safety: the
// worst outcome is an ORPHAN (created platform-side, absent from state).
//   - create error -> FAIL, never SetId.
//   - activity wait failure -> FAIL with the activityId + an audit hint (a VM may
//     exist; import or delete it before re-applying).
//   - the id is taken ONLY from the completed activity (concernedItems "vmi" or
//     the state result), never guessed by name; no id -> FAIL closed.
func createVMInstanceWith(ctx context.Context, d *schema.ResourceData, funcs vmInstanceCRUDFuncs) diag.Diagnostics {
	req, err := buildCreateVMInstanceRequest(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// VPC phase 1: the inline os_network_adapter block only supports Private
	// Backbone networks — VPC attachments go through the standalone
	// cloudtemple_public_cloud_vm_network_adapter resource. Each declared network
	// is resolved BEFORE the create POST; a network that cannot be read fails
	// closed (never create a VM against an unverifiable network).
	for _, nic := range req.NetworkInterfaces {
		network, nerr := funcs.networkRead(ctx, nic.NetworkID)
		if nerr != nil {
			return diag.Errorf("failed to verify network %s of os_network_adapter (device_index %d) before creating VM %q: %s. Refusing to create against an unverifiable network.", nic.NetworkID, nic.DeviceIndex, req.Name, nerr)
		}
		if network == nil {
			return diag.Errorf("network %s of os_network_adapter (device_index %d) could not be found before creating VM %q; refusing to create against an unverifiable network.", nic.NetworkID, nic.DeviceIndex, req.Name)
		}
		if network.VPC != nil {
			return diag.Errorf("network %s (%q) of os_network_adapter (device_index %d) is a VPC network: the inline os_network_adapter block only supports Private Backbone networks. Create the VM on a Private Backbone network and attach the VPC network with a cloudtemple_public_cloud_vm_network_adapter resource.", nic.NetworkID, network.Name, nic.DeviceIndex)
		}
	}

	activityID, err := funcs.create(ctx, req)
	if err != nil {
		return diag.Errorf("failed to create VM instance %q: %s", req.Name, err)
	}

	activity, err := funcs.waitActivity(ctx, activityID)
	if err != nil {
		return diag.Errorf(
			"VM instance %q create activity %q did not complete: %s. If a VM was created it is now ORPHANED outside the state — audit the VM instances for a recently-created %q and import it (terraform import) or delete it before re-applying.",
			req.Name, activityID, err, req.Name,
		)
	}

	// The id comes ONLY from the completed activity: the concernedItems entry of
	// type "vmi", or the terminal state result. Never guess it from the name.
	setIdFromActivityConcernedItems(d, activity, "vmi")
	if d.Id() == "" {
		setIdFromActivityState(d, activity)
	}
	if d.Id() == "" {
		return diag.Errorf(
			"VM instance %q create activity %q completed without reporting a VM id; refusing to guess it. Audit the VM instances for %q and import it if it was created.",
			req.Name, activityID, req.Name,
		)
	}

	// readAfterWrite: the just-created VM's id must never be dropped.
	return readVMInstanceInto(ctx, d, funcs, vmInstanceReadAfterWrite)
}

// readVMInstanceInto holds the testable read logic. The resource is NEVER dropped
// on an inconclusive read (E0-9):
//   - read error (403/5xx/transport) -> FAIL CLOSED, keep the resource.
//   - nil (404) in readAfterWrite -> FAIL CLOSED (eventual consistency, not a
//     deletion), keep the id.
//   - nil (404) in refresh -> confirm via a STRICT listing (200-only, complete):
//     drop ONLY if the id is absent from it; if it is still listed, or the listing
//     fails, keep the resource and error.
func readVMInstanceInto(ctx context.Context, d *schema.ResourceData, funcs vmInstanceCRUDFuncs, mode vmInstanceReadMode) diag.Diagnostics {
	id := d.Id()

	vm, err := funcs.read(ctx, id)
	if err != nil {
		return diag.Errorf(
			"failed to read VM instance %s: %s. The resource is kept in the state (a forbidden or backend error is not proof of absence); resolve the error, then refresh.",
			id, err,
		)
	}
	if vm == nil {
		if mode == vmInstanceReadAfterWrite {
			return diag.Errorf(
				"VM instance %s was just written but is not yet readable (eventual consistency); the resource is kept in the state with its id. Re-run terraform apply/refresh to populate its attributes.",
				id,
			)
		}
		// Refresh path: confirm the absence with an authoritative strict listing
		// before dropping the resource.
		vms, lerr := funcs.listStrict(ctx)
		if lerr != nil {
			return diag.Errorf(
				"VM instance %s could not be read and its deletion could not be confirmed by a strict listing: %s. The resource is kept in the state; resolve the error, then refresh.",
				id, lerr,
			)
		}
		for _, listed := range vms {
			if listed != nil && sameUUID(listed.ID, id) {
				return diag.Errorf(
					"VM instance %s could not be read but is still listed: refusing to drop it from the state (possible access restriction).",
					id,
				)
			}
		}
		// Deletion confirmed by the complete strict listing.
		d.SetId("")
		return nil
	}

	// A 200 body carrying a different id is a contract violation: fail closed
	// rather than adopt a VM that is not the one we asked for. UUIDs are compared
	// case-insensitively so an import written with upper-case hex is not rejected
	// against a canonicalised lower-case id.
	if !strings.EqualFold(vm.ID, id) {
		return diag.Errorf("VM instance %s read returned a different id %q; refusing to adopt it", id, vm.ID)
	}

	diags := setVMInstanceState(d, vm, mode)
	if diags.HasError() {
		return diags
	}
	return append(diags, setVMInstanceOSDisk(ctx, d, funcs, id)...)
}

// setVMInstanceOSDisk enriches the state with the VM's system (primary) disk —
// the FULL os_disk block is always written from the API view, so a partially
// declared config block (only size_gb) can never wipe the Computed siblings.
// The disk list is fetched fresh; a failure fails closed (the VM itself is
// kept, but the read errors — a forbidden/broken disk listing is surfaced, not
// silently ignored).
func setVMInstanceOSDisk(ctx context.Context, d *schema.ResourceData, funcs vmInstanceCRUDFuncs, vmID string) diag.Diagnostics {
	disks, err := funcs.listDisks(ctx, vmID)
	if err != nil {
		return diag.Errorf("VM instance %s: failed to read its disks to populate os_disk: %s", vmID, err)
	}
	var primary *client.PublicCloudVMDisk
	for _, dk := range disks {
		if dk != nil && dk.IsPrimary {
			primary = dk
			break
		}
	}
	if primary == nil {
		// No primary disk reported (e.g. a never-booted template edge case): clear
		// any stale os_disk block rather than keep a previously-read primary.
		sw := newStateWriter(d)
		sw.set("os_disk", []map[string]interface{}{})
		return sw.diags
	}

	sw := newStateWriter(d)
	sw.set("os_disk", []map[string]interface{}{{
		"id":           primary.ID,
		"size_gb":      primary.SizeGb,
		"storage_type": primary.StorageType,
		"position":     primary.Position,
		"is_primary":   primary.IsPrimary,
	}})
	return sw.diags
}

// setVMInstanceState writes the API view of the VM into the resource state. The
// immutable ids (az/template/family) are reconciled to their API values to catch
// an out-of-band replacement; os_network_adapter and cloud_init are not returned
// by the API and are ForceNew, so they are left untouched (kept from config).
func setVMInstanceState(d *schema.ResourceData, vm *client.PublicCloudVMInstance, mode vmInstanceReadMode) diag.Diagnostics {
	sw := newStateWriter(d)
	sw.set("name", vm.Name)
	sw.set("cpu", vm.VCPU)
	sw.set("memory", vm.RAMGb)
	sw.set("status", vm.Status)
	// Reconcile power_state from the live status ONLY on a refresh, to surface an
	// out-of-band power change. Right after a write the status can lag the
	// just-enacted transition, and power_state is the user's declared value — so it
	// is not overwritten from a possibly-transient status on the read-after-write.
	if mode == vmInstanceReadForRefresh {
		sw.set("power_state", powerStateFromStatus(vm.Status))
	}
	sw.set("disks_size_gb", vm.DisksSizeGb)
	sw.set("guest_tools_installed", vm.GuestToolsInstalled)
	sw.set("availability_zone_id", vm.AZ.ID)
	sw.set("availability_zone_name", vm.AZ.Name)
	sw.set("template_id", vm.Template.ID)
	sw.set("template_name", vm.Template.Name)
	sw.set("instance_family_id", vm.InstanceFamily.ID)
	sw.set("instance_family_name", vm.InstanceFamily.Name)
	if vm.BackupPolicy != nil {
		sw.set("backup_policy_id", vm.BackupPolicy.ID)
		sw.set("backup_policy_name", vm.BackupPolicy.Name)
	}
	sw.set("created_at", vm.CreatedAt)
	sw.set("updated_at", vm.UpdatedAt)
	return sw.diags
}

// powerStateFromStatus maps the API status to the declarative power_state. Only a
// definitively "stopped" VM is reported as off; any other status (running, or a
// transitional/unknown state) is reported as on, so the resize precondition
// (power_state == off) is never satisfied by a VM that is not fully stopped.
func powerStateFromStatus(status string) string {
	if strings.EqualFold(status, "stopped") {
		return "off"
	}
	return "on"
}

// vmUpdateOp is one step of an in-place update, executed in slice order.
type vmUpdateOp int

const (
	vmOpPatch vmUpdateOp = iota
	vmOpStop
	vmOpResize
	vmOpExtendOSDisk
	vmOpStart
)

func (op vmUpdateOp) String() string {
	switch op {
	case vmOpPatch:
		return "metadata update"
	case vmOpStop:
		return "stop"
	case vmOpResize:
		return "resize"
	case vmOpExtendOSDisk:
		return "system disk extend"
	case vmOpStart:
		return "start"
	default:
		return "unknown"
	}
}

// planVMInstanceUpdate is the pure ordering of an in-place update within a single
// apply. The order is deterministic and state-safety-critical: a stop MUST precede
// a resize (the resize requires a stopped VM) and a start MUST come LAST (after any
// resize), so a resize never runs against a running VM and the VM ends in its
// declared power state.
func planVMInstanceUpdate(metadataChanged, resizing, osDiskExtending bool, oldPS, newPS string) []vmUpdateOp {
	var ops []vmUpdateOp
	if metadataChanged {
		ops = append(ops, vmOpPatch)
	}
	if oldPS == "on" && newPS == "off" {
		ops = append(ops, vmOpStop)
	}
	if resizing {
		ops = append(ops, vmOpResize)
	}
	// The system-disk extend, like a resize, requires a stopped VM: it runs after
	// the resize and before any start, so it never targets a running VM.
	if osDiskExtending {
		ops = append(ops, vmOpExtendOSDisk)
	}
	if oldPS == "off" && newPS == "on" {
		ops = append(ops, vmOpStart)
	}
	return ops
}

// executeVMInstanceUpdate runs the planned ops in order through the injected
// seams, surfacing a failed activity as an error. Separating this from the plan
// keeps the ordering purely testable and the execution order verifiable with
// recording fakes.
func executeVMInstanceUpdate(ctx context.Context, id string, plan []vmUpdateOp, funcs vmInstanceCRUDFuncs, patchReq *client.PatchVMInstanceRequest, resizeReq *client.ResizeVMInstanceRequest, osDiskSize int) error {
	for _, op := range plan {
		var err error
		switch op {
		case vmOpPatch:
			err = runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.patch(ctx, id, patchReq) })
		case vmOpStop:
			err = runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.stop(ctx, id) })
		case vmOpResize:
			err = runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.resize(ctx, id, resizeReq) })
		case vmOpExtendOSDisk:
			err = runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.extendSystem(ctx, id, osDiskSize) })
		case vmOpStart:
			err = runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.start(ctx, id) })
		}
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}
	return nil
}

// updateVMInstanceWith holds the testable update orchestration. It computes the
// deterministic op plan (metadata -> stop -> resize -> start), builds the request
// bodies for only the changed dimensions, and executes the plan in order.
func updateVMInstanceWith(ctx context.Context, d *schema.ResourceData, funcs vmInstanceCRUDFuncs) diag.Diagnostics {
	id := d.Id()

	metadataChanged := d.HasChange("name") || d.HasChange("backup_policy_id")
	resizing := d.HasChange("cpu") || d.HasChange("memory")
	oldPS, newPS := d.GetChange("power_state")

	var patchReq *client.PatchVMInstanceRequest
	if metadataChanged {
		patchReq = &client.PatchVMInstanceRequest{}
		if d.HasChange("name") {
			n := d.Get("name").(string)
			patchReq.Name = &n
		}
		if d.HasChange("backup_policy_id") {
			b := d.Get("backup_policy_id").(string)
			patchReq.BackupPolicyID = &b
		}
	}

	var resizeReq *client.ResizeVMInstanceRequest
	if resizing {
		resizeReq = &client.ResizeVMInstanceRequest{}
		if d.HasChange("cpu") {
			cpu := d.Get("cpu").(int)
			resizeReq.CPU = &cpu
		}
		if d.HasChange("memory") {
			mem := d.Get("memory").(int)
			resizeReq.Memory = &mem
		}
	}

	// LEAF-only detection: HasChange("os_disk") would also fire on a refresh of a
	// Computed sibling (id, storage_type, ...) and trigger an extend the user
	// never asked for.
	osDiskExtending := d.HasChange("os_disk.0.size_gb")
	osDiskSize := d.Get("os_disk.0.size_gb").(int)
	if osDiskExtending && osDiskSize == 0 {
		// Block removed / no concrete target size: nothing to extend to.
		osDiskExtending = false
	}

	plan := planVMInstanceUpdate(metadataChanged, resizing, osDiskExtending, oldPS.(string), newPS.(string))
	if err := executeVMInstanceUpdate(ctx, id, plan, funcs, patchReq, resizeReq, osDiskSize); err != nil {
		return diag.Errorf("failed to update VM instance %s: %s", id, err)
	}

	return readVMInstanceInto(ctx, d, funcs, vmInstanceReadAfterWrite)
}

// deleteVMInstanceWith holds the testable delete orchestration.
func deleteVMInstanceWith(ctx context.Context, d *schema.ResourceData, funcs vmInstanceCRUDFuncs) diag.Diagnostics {
	if err := runVMInstanceActivity(ctx, funcs.waitActivity, func() (string, error) { return funcs.del(ctx, d.Id()) }); err != nil {
		return diag.Errorf("failed to delete VM instance %s: %s", d.Id(), err)
	}
	return nil
}

// runVMInstanceActivity issues an async write and waits for its activity to reach
// a terminal state, surfacing a failed activity as an error.
func runVMInstanceActivity(ctx context.Context, wait func(ctx context.Context, activityID string) (*client.Activity, error), do func() (string, error)) error {
	activityID, err := do()
	if err != nil {
		return err
	}
	_, err = wait(ctx, activityID)
	return err
}

// buildCreateVMInstanceRequest maps the resource config into the create body. It
// deliberately never sets disks[]: the resource does not create disks (the
// template provides the system disk).
func buildCreateVMInstanceRequest(d *schema.ResourceData) (*client.CreateVMInstanceRequest, error) {
	req := &client.CreateVMInstanceRequest{
		Name:               d.Get("name").(string),
		AvailabilityZoneID: d.Get("availability_zone_id").(string),
		TemplateID:         d.Get("template_id").(string),
		InstanceFamilyID:   d.Get("instance_family_id").(string),
		CPU:                d.Get("cpu").(int),
		Memory:             d.Get("memory").(int),
		BackupPolicyID:     d.Get("backup_policy_id").(string),
		PowerState:         d.Get("power_state").(string),
		NetworkInterfaces:  expandVMInstanceNICs(d.Get("os_network_adapter").([]interface{})),
	}
	req.CloudInit = expandVMInstanceCloudInit(d.Get("cloud_init").(map[string]interface{}))
	return req, nil
}

func expandVMInstanceNICs(raw []interface{}) []client.CreateVMInstanceNIC {
	nics := make([]client.CreateVMInstanceNIC, 0, len(raw))
	for _, r := range raw {
		m := r.(map[string]interface{})
		nic := client.CreateVMInstanceNIC{
			DeviceIndex: m["device_index"].(int),
			NetworkID:   m["network_id"].(string),
		}
		if ip, ok := m["ip_address"].(string); ok && ip != "" {
			nic.IPAddress = ip
		}
		nics = append(nics, nic)
	}
	return nics
}

func expandVMInstanceCloudInit(raw map[string]interface{}) *client.CreateVMInstanceCloudInit {
	if len(raw) == 0 {
		return nil
	}
	ci := &client.CreateVMInstanceCloudInit{}
	if v, ok := raw["cloud_config"].(string); ok {
		ci.CloudConfig = v
	}
	if v, ok := raw["network_config"].(string); ok {
		ci.NetworkConfig = v
	}
	return ci
}
