package provider

import (
	"context"
	"encoding/base64"
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
		Description: "Manages a Public Cloud VM instance. Creation, resize, power transitions and deletion are asynchronous (tracked through the Shiva Activities service). The system disk is provided by the image and is not created here; data disks and additional network adapters are managed by their own resources.",

		CreateContext: resourcePublicCloudVMInstanceCreate,
		ReadContext:   resourcePublicCloudVMInstanceRead,
		UpdateContext: resourcePublicCloudVMInstanceUpdate,
		DeleteContext: resourcePublicCloudVMInstanceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		// v0 -> v1: the `template_id`/`template_name` attributes were renamed to
		// `image_id`/`image_name` (the "template" catalogue object was renamed to
		// "image"). The underlying value (a catalogue UUID / name) is unchanged, so
		// the upgrader is a pure state-key rename and never forces a replacement.
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourcePublicCloudVMInstanceResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: migratePublicCloudVMInstanceStateV0toV1,
				Version: 0,
			},
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
				Description:  "The amount of RAM in GiB. Mutable via resize, which requires `power_state = \"off\"`.",
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
			"image_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the OS image the VM is created from. Immutable.",
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
				Description: "The network interfaces attached at creation. Both Private Backbone and VPC networks are supported, so a VPC-only VM can be declared here. Immutable (`ForceNew`): changing an interface's `network_id` or `ip_address` REPLACES the VM — relocate an existing adapter with a `cloudtemple_public_cloud_vm_network_adapter` resource instead. Additional adapters beyond creation are also managed by that resource.",
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
							Description:  "The fixed IPv4 address to assign, registered as a static IP on the VPC private network. Requires `network_id` to reference a VPC network: the platform silently ignores it on a Private Backbone network, so setting it there is rejected at apply. It is also rejected when the address is ALREADY registered on the target VPC private network: the platform does not refuse that case — it creates the VM, reports success and silently registers nothing — so the collision is refused before the create (the authoritative check) and reported earlier, at plan, when the plan carries the VM's identity (an in-place plan, or the first pass of a replacement plan). A fresh create and the create half of a replacement are planned without one and cannot tell the VM's own registration from another machine's, so they are left to the pre-create check. When omitted on a VPC network, the platform auto-assigns an address. Write-only: it is never read back (the registration is addressable only by MAC on the VPC plane).",
						},
					},
				},
			},
			"cloud_init": {
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "The cloud-init configuration applied at creation (keys `cloud_config` and/or `network_config`), as plain YAML — the provider base64-encodes it for the API. Immutable and not readable back, so it is not reconciled on refresh.",
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
			"disks_size_gib": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The total size of the VM's disks (system + data) in GiB.",
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
			"image_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the OS image.",
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
				Description: "The system (primary) disk of the VM, provided by the image. Declare the block with `size_gib` to grow it (grow-only; requires the VM to be stopped). Not settable at creation — the image's size is used. Data disks are managed by the separate disk resource.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the system disk."},
						"size_gib": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntAtLeast(1),
							Description:  "The size of the system disk in GiB. Grow-only; increasing it extends the system disk, which requires the VM to be stopped. When omitted, the current size is kept.",
						},
						// Deprecated spelling of size_gib, kept for the API compatibility
						// window (issue #524). It is NOT a different unit: the VM Instances
						// API renamed the field without changing the value, so the two carry
						// the same number. Removal is tracked by issue #525.
						"size_gb": {
							Type:         schema.TypeInt,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IntAtLeast(1),
							Deprecated:   "Use size_gib instead. The value is identical — the VM Instances API renamed the field to match the binary unit it always used. This attribute will be removed in a future major version.",
							Description:  "Deprecated: use `size_gib`, which carries the same value. The size of the system disk in GiB.",
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

// customizeVMInstanceDiff is the production CustomizeDiff: the IPAM view of the
// plan-time collision check comes from the configured client (nil in unit tests
// that drive Resource.Diff with a nil meta, where the hook degrades to a no-op).
func customizeVMInstanceDiff(ctx context.Context, d *schema.ResourceDiff, meta any) error {
	return customizeVMInstanceDiffWith(inlineIPConflictOrNil(meta, publicCloudInlineIPConflict))(ctx, d, meta)
}

// customizeVMInstanceDiffWith enforces the resize precondition at PLAN time: cpu and
// memory can only change while the VM is declared stopped. This is checked ONLY
// on an existing resource — on create, cpu/memory necessarily "change" from their
// zero value and an `on` power_state is the legitimate boot-at-create case.
//
// conflictOf is the IPAM view of the plan-time collision check. It is injected so a
// test can drive the REAL resource through Resource.SimpleDiff — the exact entry point
// the gRPC PlanResourceChange uses — with a recorded IPAM answer and no network.
func customizeVMInstanceDiffWith(conflictOf inlineIPConflictFunc) schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
		// Plan-time IPAM collision check; see inlineAdapterIPCollisionDiff. Advisory:
		// the create preflight remains the authoritative gate.
		if err := inlineAdapterIPCollisionDiff(conflictOf)(ctx, d, meta); err != nil {
			return err
		}
		exists := d.Id() != ""
		// The os_disk size cannot be set at CREATE: the VM is created with the
		// image's system disk size, and create never extends it — so a configured
		// value that differs from the image would produce an inconsistent result.
		// Detected on the RAW config (never the diff/state: the attribute is
		// Optional+Computed, so d.Get would report a computed value as "set").
		declared := osDiskSizeDeclaredInRawConfig(d.GetRawConfig())
		// The two spellings carry the same value, so declaring both can only express
		// a contradiction (or hide one). Rejected at plan time, before any write.
		if declared.both() {
			return fmt.Errorf("os_disk: set either size_gib or the deprecated size_gb, not both — they are the same value under two names. Keep size_gib")
		}
		if !exists {
			if declared.any() {
				return fmt.Errorf("os_disk.size_gib cannot be set when creating the VM: the image's system disk size is used at creation. Omit it, then set it in a later apply (with power_state = \"off\") to grow the system disk")
			}
		} else if attr, changed := osDiskChangedSizeAttr(d); changed {
			// Grow-only + requires-off, validated on the LEAF only: a refresh of a
			// Computed sibling (id, storage_type, ...) must never look like an extend.
			o, n := d.GetChange(attr)
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
}

// osDiskSizeDeclaration reports which of the two os_disk size spellings the raw
// config declares. Both carry the same value (issue #524), so which one the user
// wrote decides which leaf the plan-time rules read.
type osDiskSizeDeclaration struct {
	Gib        bool // size_gib explicitly set
	Deprecated bool // size_gb explicitly set
	Unknown    bool // the os_disk block itself is unknown at plan time
}

// any reports whether a size is declared at all — including the unknown case,
// which create cannot honour either.
func (o osDiskSizeDeclaration) any() bool { return o.Gib || o.Deprecated || o.Unknown }

// both reports the contradictory case: the same size declared under two names.
// An unknown block is NOT "both": nothing can be read from it, so it must not
// trigger the conflict error.
func (o osDiskSizeDeclaration) both() bool { return o.Gib && o.Deprecated }

// osDiskSizeDeclaredInRawConfig inspects the raw config for an os_disk block
// declaring a size under either spelling (null-, absent- and unknown-safe). It
// reads the RAW config because both size attributes are Optional+Computed — the
// diff/state view cannot tell an explicit value from a computed one.
func osDiskSizeDeclaredInRawConfig(raw cty.Value) osDiskSizeDeclaration {
	var out osDiskSizeDeclaration
	if raw.IsNull() || !raw.Type().IsObjectType() || !raw.Type().HasAttribute("os_disk") {
		return out
	}
	osd := raw.GetAttr("os_disk")
	if osd.IsNull() {
		return out
	}
	// A declared-but-unknown os_disk cannot be inspected — and create cannot
	// honour a size that only resolves later. Reject conservatively.
	if !osd.IsKnown() {
		out.Unknown = true
		return out
	}
	if !osd.CanIterateElements() {
		return out
	}
	for it := osd.ElementIterator(); it.Next(); {
		_, el := it.Element()
		if el.IsNull() || !el.Type().IsObjectType() {
			continue
		}
		if el.Type().HasAttribute("size_gib") {
			if sg := el.GetAttr("size_gib"); !sg.IsNull() {
				out.Gib = true
			}
		}
		if el.Type().HasAttribute("size_gb") {
			if sg := el.GetAttr("size_gb"); !sg.IsNull() {
				out.Deprecated = true
			}
		}
	}
	return out
}

// osDiskChangedSizeAttr returns the os_disk size leaf that actually changed, so
// that the grow-only rules and the update path read the spelling the user
// drives. The current spelling wins when both report a change (a state written
// before the rename can make the deprecated leaf move on its own).
func osDiskChangedSizeAttr(d *schema.ResourceDiff) (string, bool) {
	if d.HasChange(osDiskSizeGibAttr) {
		return osDiskSizeGibAttr, true
	}
	if d.HasChange(osDiskSizeGbAttr) {
		return osDiskSizeGbAttr, true
	}
	return "", false
}

// The two os_disk size leaves, as addressed in the diff/state.
const (
	osDiskSizeGibAttr = "os_disk.0.size_gib"
	osDiskSizeGbAttr  = "os_disk.0.size_gb"
)

// vmInstanceOSDiskChangeCheck is the pure os_disk size change rule on an
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
		return fmt.Errorf("os_disk.size_gib can only be changed while power_state = \"off\" (extending the system disk requires a stopped VM); set power_state = \"off\" in the same change, then power the VM back on in a subsequent apply")
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
	// listStaticIPs reads every static IP registered on a VPC private network, so
	// the create can refuse an address that is already taken. STRICT on purpose: a
	// truncated body must surface as an error, never read as an empty network — see
	// rejectInlineAdapterIPAlreadyRegistered.
	listStaticIPs func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error)
}

func vmInstanceWaiterOptions(ctx context.Context) *client.WaiterOptions {
	o := getWaiterOptions(ctx)
	o.NotFoundRetries = vmInstanceActivityNotFoundRetries
	return o
}

// publicCloudInlineIPConflict builds the IPAM-collision checker for VM Instances.
func publicCloudInlineIPConflict(c *client.Client) inlineIPConflictFunc {
	return staticIPConflictChecker(
		func(ctx context.Context, networkID string) (string, error) {
			network, err := c.PublicCloudVM().Network().Read(ctx, networkID)
			if err != nil {
				return "", err
			}
			if network == nil || network.VPC == nil {
				return "", nil
			}
			return network.VPC.PrivateNetwork.ID, nil
		},
		func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
		},
	)
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
		listStaticIPs: func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
			return c.VPC().StaticIP().ListStrict(ctx, privateNetworkID)
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

	// Network preflight. Each declared network is resolved BEFORE the create POST;
	// a network that cannot be read fails closed (never create a VM against an
	// unverifiable network).
	//
	// VPC networks ARE supported here. The former phase-1 refusal was a provider
	// scoping decision, not an API limitation, and it has been retired against live
	// evidence (DEV 2026-08-20): the create endpoint accepts a VPC networkId, the
	// resulting adapter comes up `type="vpc"`, and networkInterfaces[].ipAddress is
	// honoured — see internal/client/vpc_vm_create_live_probe_test.go.
	//
	// What the preflight now enforces instead is the ip_address precondition. The
	// same live probe measured that ipAddress on a Private Backbone network is
	// SILENTLY IGNORED: no static IP is registered and no error is returned. Left
	// unchecked, a user's chosen address would be dead config they believe took
	// effect, so it is rejected up front — the same polarity as the two Compute
	// standalone adapter resources (see validateIPAddressTargetsVPC).
	for _, nic := range req.NetworkInterfaces {
		network, nerr := funcs.networkRead(ctx, nic.NetworkID)
		if nerr != nil {
			return diag.Errorf("failed to verify network %s of os_network_adapter (device_index %d) before creating VM %q: %s. Refusing to create against an unverifiable network.", nic.NetworkID, nic.DeviceIndex, req.Name, nerr)
		}
		if network == nil {
			return diag.Errorf("network %s of os_network_adapter (device_index %d) could not be found before creating VM %q; refusing to create against an unverifiable network.", nic.NetworkID, nic.DeviceIndex, req.Name)
		}
		if nic.IPAddress != "" && network.VPC == nil {
			return diag.Errorf("ip_address %q is set on os_network_adapter (device_index %d) but network %s (%q) is not a VPC network: a fixed IPv4 address is only honoured on a VPC network (the platform silently ignores it elsewhere) — remove ip_address, or target a VPC network.", nic.IPAddress, nic.DeviceIndex, nic.NetworkID, network.Name)
		}
		// IPAM collision, checked before the create for the reason documented on
		// rejectInlineAdapterIPAlreadyRegistered: an address already registered to
		// someone else is NOT refused by the platform — the VM is created, success is
		// reported, and the address is silently not registered, while the state records
		// it. Nothing detects that afterwards, because ip_address is write-only.
		//
		// Fail closed on a read error: an unreadable listing must not be read as "free".
		if nic.IPAddress != "" && network.VPC != nil && funcs.listStaticIPs != nil {
			pnID := network.VPC.PrivateNetwork.ID
			if pnID == "" {
				return diag.Errorf("network %s (%q) is VPC-backed but exposes no private-network id, so the ip_address %q of os_network_adapter (device_index %d) cannot be checked for a collision; refusing rather than requesting an address the platform may silently decline to register.", nic.NetworkID, network.Name, nic.IPAddress, nic.DeviceIndex)
			}
			registered, err := funcs.listStaticIPs(ctx, pnID)
			if err != nil {
				return diag.Errorf("failed to verify whether ip_address %q is already registered on the VPC private network %s of network %s (os_network_adapter device_index %d): %s", nic.IPAddress, pnID, nic.NetworkID, nic.DeviceIndex, err)
			}
			for _, existing := range registered {
				if existing == nil || existing.IPAddress != nic.IPAddress {
					continue
				}
				return diag.Errorf("os_network_adapter (device_index %d) requests ip_address %q, which is ALREADY registered on the VPC private network of network %s%s. The platform does not refuse this: it would create the VM, report success, and silently leave the adapter with no static IP while Terraform recorded %q. Choose a free address.%s", nic.DeviceIndex, nic.IPAddress, nic.NetworkID, describeStaticIPHolder(existing), nic.IPAddress, inlineAdapterIPReplacementHint)
			}
		}
	}

	activityID, err := funcs.create(ctx, req)
	if err != nil {
		return diag.Errorf("failed to create VM instance %q: %s", req.Name, err)
	}

	activity, err := funcs.waitActivity(ctx, activityID)
	if err != nil {
		// Two planes can be left behind by a create that fails after starting.
		// The compute plane is the VM itself. The IPAM plane is a VPC static-IP
		// registration: a successful delete reclaims it (verified live), but a
		// create that registered the address and then failed leaves it behind, and
		// the provider cannot safely reclaim it — it has no positive evidence of
		// ownership for an address it may never have chosen (auto-assignment). So
		// the diagnostic names both planes instead of silently leaving one out.
		return diag.Errorf(
			"VM instance %q create activity %q did not complete: %s. If a VM was created it is now ORPHANED outside the state — audit the VM instances for a recently-created %q and import it (terraform import) or delete it before re-applying. If any os_network_adapter targeted a VPC network, also audit that VPC private network's static IPs for a registration left behind by this failed create.",
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
// declared config block (only a size) can never wipe the Computed siblings.
// Both size spellings are written with the SAME value: the deprecated one must
// never be left holding a stale number while the window is open (issue #524).
// Consequence, deliberately accepted: growing the disk through size_gib leaves
// the deprecated leaf showing its previous value in that one plan, since an
// Optional+Computed attribute absent from the config keeps its state value in
// the diff. The apply reconciles both, so the state is never wrong — only that
// intermediate plan is incomplete. Neither SetNew nor SetNewComputed can fix it
// (both reject a nested path: helper/schema checkKey looks the key up in the
// root schema map), and rebuilding the whole os_disk block in CustomizeDiff
// would risk wiping the very Computed siblings this function protects.
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
		// No primary disk reported (e.g. a never-booted image edge case): clear
		// any stale os_disk block rather than keep a previously-read primary.
		sw := newStateWriter(d)
		sw.set("os_disk", []map[string]interface{}{})
		return sw.diags
	}

	sw := newStateWriter(d)
	sw.set("os_disk", []map[string]interface{}{{
		"id":           primary.ID,
		"size_gib":     primary.SizeGib,
		"size_gb":      primary.SizeGib,
		"storage_type": primary.StorageType,
		"position":     primary.Position,
		"is_primary":   primary.IsPrimary,
	}})
	return sw.diags
}

// setVMInstanceState writes the API view of the VM into the resource state. The
// immutable ids (az/image/family) are reconciled to their API values to catch
// an out-of-band replacement; os_network_adapter and cloud_init are not returned
// by the API and are ForceNew, so they are left untouched (kept from config).
func setVMInstanceState(d *schema.ResourceData, vm *client.PublicCloudVMInstance, mode vmInstanceReadMode) diag.Diagnostics {
	sw := newStateWriter(d)
	sw.set("name", vm.Name)
	sw.set("cpu", vm.VCPU)
	sw.set("memory", vm.RAMGib)
	sw.set("status", vm.Status)
	// Reconcile power_state from the live status ONLY on a refresh, to surface an
	// out-of-band power change. Right after a write the status can lag the
	// just-enacted transition, and power_state is the user's declared value — so it
	// is not overwritten from a possibly-transient status on the read-after-write.
	if mode == vmInstanceReadForRefresh {
		sw.set("power_state", powerStateFromStatus(vm.Status))
	}
	sw.set("disks_size_gib", vm.DisksSizeGib)
	sw.set("guest_tools_installed", vm.GuestToolsInstalled)
	sw.set("availability_zone_id", vm.AZ.ID)
	sw.set("availability_zone_name", vm.AZ.Name)
	sw.set("image_id", vm.Image.ID)
	sw.set("image_name", vm.Image.Name)
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
	// never asked for. Either spelling drives the extend, the current one first.
	osDiskSizeAttr := osDiskSizeGibAttr
	osDiskExtending := d.HasChange(osDiskSizeGibAttr)
	if !osDiskExtending && d.HasChange(osDiskSizeGbAttr) {
		osDiskSizeAttr = osDiskSizeGbAttr
		osDiskExtending = true
	}
	osDiskSize := d.Get(osDiskSizeAttr).(int)
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
// image provides the system disk).
func buildCreateVMInstanceRequest(d *schema.ResourceData) (*client.CreateVMInstanceRequest, error) {
	req := &client.CreateVMInstanceRequest{
		Name:               d.Get("name").(string),
		AvailabilityZoneID: d.Get("availability_zone_id").(string),
		ImageID:            d.Get("image_id").(string),
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

// expandVMInstanceCloudInit maps the cloud_init config into the create body.
// The API only accepts base64-encoded payloads (verified live: a raw YAML is
// rejected with "Must be a valid base64-encoded string"), so the user writes
// plain YAML and the provider encodes it — always, never conditionally, so a
// payload that happens to look like base64 is never double-interpreted.
func expandVMInstanceCloudInit(raw map[string]interface{}) *client.CreateVMInstanceCloudInit {
	if len(raw) == 0 {
		return nil
	}
	ci := &client.CreateVMInstanceCloudInit{}
	if v, ok := raw["cloud_config"].(string); ok && v != "" {
		ci.CloudConfig = base64.StdEncoding.EncodeToString([]byte(v))
	}
	if v, ok := raw["network_config"].(string); ok && v != "" {
		ci.NetworkConfig = base64.StdEncoding.EncodeToString([]byte(v))
	}
	return ci
}

// resourcePublicCloudVMInstanceResourceV0 returns the V0 schema, used only to
// migrate state written before the `template` -> `image` rename. It declares
// just the two attributes whose keys changed; every other attribute is carried
// through unchanged by migratePublicCloudVMInstanceStateV0toV1.
//
// The minimal-V0 pattern (declare only the changed keys, not a frozen copy of
// the full schema) is safe here and matches the repository idiom
// (migrateVirtualMachineStateV0toV1): this is an SDKv2 provider, so it only
// runs under Terraform >= 0.12, whose state is JSON. On the JSON upgrade path
// the SDK hands the upgrade function the FULL raw state map regardless of this
// Type, so unrelated attributes are preserved (proven by
// TestMigratePublicCloudVMInstanceStateV0toV1). The legacy flatmap path — the
// only case where an undeclared attribute could be dropped through this Type —
// cannot apply: flatmap state exists only under Terraform <= 0.11, which cannot
// run an SDKv2 provider, and this resource is new in the 1.10.0 cycle so all of
// its state is recent v0 JSON.
func resourcePublicCloudVMInstanceResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"template_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"template_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// migratePublicCloudVMInstanceStateV0toV1 renames the state keys after the
// `template` -> `image` catalogue rename: `template_id` -> `image_id` and
// `template_name` -> `image_name`. The stored value (a catalogue UUID / name) is
// identical for the renamed object, so this is a pure key rename: an existing
// VM keeps its identity and is NOT forced to be recreated. Users must still
// rename the argument in their HCL (`template_id` -> `image_id`); the state
// migration only prevents the ForceNew replacement that the rename would
// otherwise trigger.
func migratePublicCloudVMInstanceStateV0toV1(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	if rawState == nil {
		return rawState, nil
	}
	if v, ok := rawState["template_id"]; ok {
		rawState["image_id"] = v
		delete(rawState, "template_id")
	}
	if v, ok := rawState["template_name"]; ok {
		rawState["image_name"] = v
		delete(rawState, "template_name")
	}
	return rawState, nil
}
