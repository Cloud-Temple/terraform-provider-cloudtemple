package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceOpenIaasVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Create and manage virtual machines over an Open IaaS infrastructure.",

		CreateContext: openIaasVirtualMachineCreate,
		ReadContext:   openIaasVirtualMachineRead,
		UpdateContext: openIaasVirtualMachineUpdate,
		DeleteContext: openIaasVirtualMachineDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			//In
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the virtual machine.",
				Required:    true,
			},
			"template_id": {
				Type:          schema.TypeString,
				Description:   "The template identifier.",
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"marketplace_item_id"},
				AtLeastOneOf:  []string{"template_id", "marketplace_item_id"},
			},
			"marketplace_item_id": {
				Type:          schema.TypeString,
				Description:   "The marketplace item identifier to deploy the virtual machine from.",
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"template_id"},
				AtLeastOneOf:  []string{"template_id", "marketplace_item_id"},
			},
			"storage_repository_id": {
				Type:          schema.TypeString,
				Description:   "The storage repository identifier where the virtual machine will be created. Required when `marketplace_item_id` is set.",
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"template_id"},
				RequiredWith:  []string{"marketplace_item_id"},
			},
			"cpu": {
				Type:        schema.TypeInt,
				Description: "The number of virtual CPUs. Note: Changing this value for a running VM will cause it to be powered off and back on.",
				Required:    true,
			},
			"num_cores_per_socket": {
				Type:         schema.TypeInt,
				Description:  "The number of cores per socket. Note: Changing this value for a running VM will cause it to be powered off and back on.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
			"memory": {
				Type:        schema.TypeInt,
				Description: "The amount of memory in Bytes. Note: Changing this value for a running VM will cause it to be powered off and back on.",
				Required:    true,
			},
			"power_state": {
				Type:         schema.TypeString,
				Description:  "The desired power state of the virtual machine. Available values are 'on' and 'off'.",
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
			},
			"host_id": {
				Type:         schema.TypeString,
				Description:  "The host identifier.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsUUID,
			},
			"boot_order": {
				Type: schema.TypeList,
				Description: `The boot order of the virtual machine.
Available values are 'Hard-Drive', 'DVD-Drive', and 'Network'.
Order of the elements in the list is the boot order.`,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"Hard-Drive", "DVD-Drive", "Network"}, false),
				},
			},
			"mount_iso": {
				Type:         schema.TypeString,
				Description:  "An ISO disk to mount to on the virtual machine DVD Drive.",
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"secure_boot": {
				Type:        schema.TypeBool,
				Description: "Whether to enable secure boot. Only available with UEFI boot firmware.",
				Optional:    true,
				Computed:    true,
			},
			"boot_firmware": {
				Type:        schema.TypeString,
				Description: "The boot firmware to use. Available values are 'bios' and 'uefi'.",
				Optional:    true,
				// Computed: marketplace images set it (uefi). Without it the
				// plan never converges (permanent "uefi" -> null drift).
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"bios", "uefi"}, false),
			},
			"auto_power_on": {
				Type:        schema.TypeBool,
				Description: "Whether to automatically start the virtual machine when the host boots.",
				Optional:    true,
			},
			"high_availability": {
				Type:         schema.TypeString,
				Description:  "High Availability configuration for the virtual machine (Default: disabled). Possible values are: 'disabled', 'restart' and 'best-effort'. For more informations, refer to the documentation : https://docs.cloud-temple.com/iaas_opensource/concepts#haute-disponibilit%C3%A9",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"disabled", "best-effort", "restart"}, false),
				Default:      "disabled",
			},
			"replication_policy_id": {
				Type:         schema.TypeString,
				Description:  "The ID of the replication policy to associate with the virtual machine.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IsUUID,
			},
			"backup_sla_policies": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "The IDs of the SLA policies to assign to the virtual machine.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},
			"wait_for_drivers_timeout": {
				Type:         schema.TypeInt,
				Description:  "The maximum time in seconds to wait for PV drivers to be detected after starting the VM. Set to 0 to skip waiting. Default is 30 seconds.",
				Optional:     true,
				Default:      30,
				ValidateFunc: validation.IntBetween(0, 900),
			},
			"allow_vm_restart": {
				Type: schema.TypeBool,
				Description: `Whether the user allows the povider to restart the VM if necessary operations require it. Default is true.
	The virtual machine will need to be restart when at least one of these properties are modified :
		- "cpu", "ram" or "num_cores_per_socket"
		- "os_disk.size" or "os_disk.name"
		- "os_network_adapter.mac_address"`,
				Optional: true,
				Default:  true,
			},
			"tags": {
				Type:        schema.TypeMap,
				Description: "The tags to attach to the virtual machine.",
				Optional:    true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"cloud_init": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Description: `A set of cloud-init compatible key/value used to configure the virtual machine.
					
	List of cloud-init compatible keys :
	- ` + "`cloud_config`" + `
	- ` + "`network_config`" + `

	Please note that the virtual machine must have a disk in order to use Cloud-Init.
	
	If you need more informations, please refer to the cloud-init documentation about the NoCloud datasource.

	NB : The cloud-init configuration is only triggered at virtual machine first startup and requires a cloud-init compatible NoCloud.
	For exemple, you can use this [Ubuntu Cloud Image](https://cloud-images.ubuntu.com/) and convert it to an NoCloud.
				`,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ValidateDiagFunc: validation.MapKeyMatch(regexp.MustCompile(strings.Join([]string{
					"^cloud_config$",
					"^network_config$"},
					"|")), `The following key is not allowed for cloud-init`),
			},
			"os_disk": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "The operating system disk of the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The name of the operating system disk. (Updating this property implies a disk disconnect-reconnect)",
						},
						"size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Computed:    true,
							Description: "The size of the operating system disk in bytes. (Updating this property implies a disk disconnect-reconnect)",
						},
						"connected": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Whether the disk is connected or not.",
							Deprecated:  "This property is deprecated and will be definitly removed in a future version.",
						},
						"storage_repository_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The storage repository where the operating system disk is located.",
						},

						// Out
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual disk.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the operating system disk.",
						},
					},
				},
			},
			"os_network_adapter": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "The network adapters of the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mac_address": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The MAC address of the network adapter. If not provided, the MAC address will be sourced from the template used.",
						},
						"network_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The identifier of the network to which the adapter is connected.  If not provided, the network will be sourced from the template used.",
						},
						"attached": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Deprecated:  "This property is deprecated and will be definitly removed in a future version.",
							Description: "Whether the network adapter is attached.",
						},
						"tx_checksumming": {
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
							Description: "Whether TX checksumming is enabled on the network adapter.",
						},

						// Out
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the network adapter.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network adapter.",
						},
						"mtu": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum transmission unit (MTU) of the network adapter.",
						},
					},
				},
			},

			//Out
			"machine_manager_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the machine manager (availability zone).",
			},
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the virtual machine.",
			},
			"dvd_drive": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The DVD drive of the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the ISO mounted in the DVD drive.",
						},
						"attached": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the DVD drive is attached.",
						},
					},
				},
			},
			// Deprecated: Use pv_drivers and management_agent instead. This field will be removed in a future version.
			"tools": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The tools installed on the virtual machine. Please note that the tools are only available when the virtual machine is powered on.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the tools are detected.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the tools.",
						},
					},
				},
			},
			"pv_drivers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The paravirtual (PV) drivers installed on the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the PV drivers are detected.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the PV drivers.",
						},
						"are_up_to_date": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the PV drivers are up to date.",
						},
					},
				},
			},
			"management_agent": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The management agent installed on the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the management agent is detected.",
						},
					},
				},
			},
			"operating_system_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the operating system installed on the virtual machine.",
			},
			"addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The addresses of the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv6": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"pool_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the pool to which the virtual machine belongs.",
			},
		},
		CustomizeDiff: customdiff.All(
			customdiff.ValidateChange("os_disk", func(ctx context.Context, old, new, meta any) error {
				o := len(old.([]interface{}))
				n := len(new.([]interface{}))
				if n > o && o > 0 {
					return fmt.Errorf("new os_disk blocks are not allowed if that exceeds the number of existing OS disks (%d > %d)", n, o)
				}
				return nil
			}),
			customdiff.ValidateChange("os_network_adapter", func(ctx context.Context, old, new, meta any) error {
				o := len(old.([]interface{}))
				n := len(new.([]interface{}))
				if n > o && o > 0 {
					return fmt.Errorf("new os_network_adapter blocks are not allowed if that exceeds the number of existing OS network adapters (%d > %d)", n, o)
				}
				return nil
			}),
		),
	}
}

// buildOpenIaasCloudInit maps the optional cloud_init schema attribute to the
// client payload. It returns:
//   - nil when cloud_init is absent, empty, or only carries empty values, so the
//     "cloudInit" field is OMITTED from the request (the API rejects an empty
//     cloudInit object with "cloudConfig is required");
//   - an error when cloud_init is set with a network_config but no cloud_config —
//     the API requires cloud_config whenever cloudInit is present, so we fail
//     closed with a clear message instead of forwarding a doomed request;
//   - a populated *client.CloudInit when cloud_config is set (network_config
//     optional).
func buildOpenIaasCloudInit(raw map[string]interface{}) (*client.CloudInit, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	cloudConfig, _ := raw["cloud_config"].(string)
	networkConfig, _ := raw["network_config"].(string)
	if cloudConfig == "" {
		if networkConfig != "" {
			return nil, fmt.Errorf("cloud_init requires cloud_config: set cloud_config (the API rejects a cloud-init without it)")
		}
		return nil, nil
	}
	return &client.CloudInit{CloudConfig: cloudConfig, NetworkConfig: networkConfig}, nil
}

func openIaasVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Cloud-Init to configure the virtual machine. A nil result omits the
	// "cloudInit" field entirely — the API rejects an empty cloudInit object.
	cloudInitRaw, _ := d.Get("cloud_init").(map[string]interface{})
	cloudInit, err := buildOpenIaasCloudInit(cloudInitRaw)
	if err != nil {
		return diag.FromErr(err)
	}

	osNetworkAdapters := d.Get("os_network_adapter").([]interface{})

	// Deploy from template
	if d.Get("template_id").(string) != "" {
		template, err := c.Compute().OpenIaaS().Template().Read(ctx, d.Get("template_id").(string))
		if err != nil {
			return diag.Errorf("Could not read the template : %s", err)
		}
		if template == nil {
			return diag.Errorf("Could not find template with id : %s", d.Get("template_id").(string))
		}

		if osNetworkAdapters != nil && len(osNetworkAdapters) != len(template.NetworkAdapters) {
			return diag.Errorf("the number of os_network_adapter (%d) must match the number of network adapters in the template (%d)", len(osNetworkAdapters), len(template.NetworkAdapters))
		}

		templateNetworkAdapters := make([]client.OSNetworkAdapter, len(template.NetworkAdapters))
		for i := range template.NetworkAdapters {
			osNetworkAdapter := osNetworkAdapters[i].(map[string]interface{})
			templateNetworkAdapters[i] = client.OSNetworkAdapter{
				NetworkID: osNetworkAdapter["network_id"].(string),
				MAC:       osNetworkAdapter["mac_address"].(string),
			}
		}

		activityId, err := c.Compute().OpenIaaS().VirtualMachine().Create(ctx, &client.CreateOpenIaasVirtualMachineRequest{
			Name:            d.Get("name").(string),
			TemplateID:      d.Get("template_id").(string),
			CPU:             d.Get("cpu").(int),
			Memory:          d.Get("memory").(int),
			CloudInit:       cloudInit,
			NetworkAdapters: templateNetworkAdapters,
		})
		if err != nil {
			return diag.Errorf("the virtual machine could not be created: %s", err)
		}
		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to create virtual machine, %s", err)
		}

		// Deploy from marketplace item
	} else if d.Get("marketplace_item_id").(string) != "" {
		openIaasItemInfo, _, err := c.Marketplace().Item().ReadInfo(ctx, d.Get("marketplace_item_id").(string), "open_iaas")
		if err != nil {
			return diag.Errorf("Could not read the marketplace item : %s", err)
		}
		if openIaasItemInfo == nil {
			return diag.Errorf("Could not find marketplace item info with id : %s", d.Get("marketplace_item_id").(string))
		}

		if osNetworkAdapters != nil && len(osNetworkAdapters) != len(openIaasItemInfo.NetworkAdapters) {
			return diag.Errorf("the number of os_network_adapter (%d) must match the number of network adapters in the marketplace item (%d)", len(osNetworkAdapters), len(openIaasItemInfo.NetworkAdapters))
		}

		networkData := []client.NetworkDataMapping{}
		for i, networkAdapter := range openIaasItemInfo.NetworkAdapters {
			osNetworkAdapter := osNetworkAdapters[i].(map[string]interface{})
			networkData = append(networkData, client.NetworkDataMapping{
				// networkAdapterName is the field recommended by the
				// marketplace API (sourceNetworkName is deprecated); both are
				// sent for backward compatibility, the name takes priority.
				NetworkAdapterName:   networkAdapter.Name,
				SourceNetworkName:    networkAdapter.NetworkName,
				DestinationNetworkId: osNetworkAdapter["network_id"].(string),
			})
		}

		activityId, err := c.Marketplace().Item().DeployOpenIaasItem(ctx, &client.MarketplaceOpenIaasDeployementRequest{
			ID:                  d.Get("marketplace_item_id").(string),
			Name:                d.Get("name").(string),
			StorageRepositoryID: d.Get("storage_repository_id").(string),
			NetworkData:         networkData,
			CloudInit:           cloudInit,
		})
		if err != nil {
			return diag.Errorf("the virtual machine could not be created from marketplace item: %s", err)
		}
		activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		setIdFromActivityState(d, activity)
		if err != nil {
			return diag.Errorf("failed to create virtual machine from marketplace item, %s", err)
		}
	}

	disks, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, &client.OpenIaaSVirtualDiskFilter{
		VirtualMachineID: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to read os_disks, %s", err)
	}
	if disks == nil {
		return diag.Errorf("could not list disks of virtual machine, %s", err)
	}

	osDisks := helpers.UpdateNestedMapItems(d, helpers.FlattenOpenIaaSOSDisksData(disks, d.Id()), "os_disk")
	if err := d.Set("os_disk", osDisks); err != nil {
		return diag.FromErr(err)
	}

	networkAdapters, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &client.OpenIaaSNetworkAdapterFilter{
		VirtualMachineID: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to read os_network_adapters, %s", err)
	}
	if networkAdapters == nil {
		return diag.Errorf("could not list network adapters of virtual machine, %s", err)
	}

	osNetworkAdapters = helpers.UpdateNestedMapItems(d, helpers.FlattenOpenIaaSOSNetworkAdaptersData(networkAdapters), "os_network_adapter")
	if err := d.Set("os_network_adapter", osNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	// Assign the requested SLA policies. Skipped entirely when none are
	// requested: a freshly created VM has nothing to assign or clear, and an
	// empty-list assign creates a backup "assign job" activity that the
	// platform can leave stuck in "Waiting" indefinitely, hanging the Create
	// until timeout (live-confirmed on the recette tenant, #306). The VM id is
	// already committed via setIdFromActivityState above, before this step.
	slaPolicies := []string{}
	for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
		slaPolicies = append(slaPolicies, policy.(string))
	}
	if err := assignBackupSLAPoliciesIfAny(ctx, c, d.Id(), slaPolicies); err != nil {
		return diag.Errorf("failed to assign policies to virtual machine, %s", err)
	}

	return openIaasVirtualMachineUpdate(ctx, d, meta)
}

// assignBackupSLAPoliciesIfAny assigns the given backup SLA policies to the
// virtual machine and waits for the assign activity to complete. It is a NO-OP
// when policies is empty: assigning an empty list to a freshly created VM has
// no effect, and the platform can leave such an activity stuck in "Waiting",
// which previously hung the Create until timeout — and, if hard-cancelled,
// could leave a VM created platform-side but absent from the Terraform state
// (#306). Callers must commit the VM id before invoking it.
func assignBackupSLAPoliciesIfAny(ctx context.Context, c *client.Client, virtualMachineID string, policies []string) error {
	if len(policies) == 0 {
		return nil
	}
	activityId, err := c.Backup().OpenIaaS().Policy().Assign(ctx, &client.BackupOpenIaasAssignPolicyRequest{
		VirtualMachineId: virtualMachineID,
		PolicyIds:        policies,
	})
	if err != nil {
		return err
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	return err
}

func openIaasVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Get the virtual machine by its ID
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("the virtual machine could not be read: %s", err)
	}
	if vm == nil {
		// The API answers 403 for unknown AND forbidden ids alike, and the
		// client maps both to nil: the VM deletion is only accepted after
		// the strict tenant listing (complete 200 required) confirms the
		// absence (#275 doctrine, FF-5).
		vms, err := c.Compute().OpenIaaS().VirtualMachine().ListStrict(ctx, &client.OpenIaaSVirtualMachineFilter{})
		if err != nil {
			return diag.Errorf("virtual machine %s could not be read and its deletion could not be confirmed: %s", d.Id(), err)
		}
		for _, listed := range vms {
			if listed != nil && listed.ID == d.Id() {
				return diag.Errorf("virtual machine %s could not be read but is still listed: refusing to drop it from the state (possible access restriction)", d.Id())
			}
		}
		// Deletion confirmed by independent strict reads.
		d.SetId("")
		return nil
	}

	// Normalize the power state to be consistent with the input
	switch vm.PowerState {
	case "Running":
		vm.PowerState = "on"
	case "Halted":
		vm.PowerState = "off"
	case "Paused":
		vm.PowerState = "off"
	default:
		return diag.Errorf("unknown power state %q", vm.PowerState)
	}

	// Map the data using the helper function
	vmData := helpers.FlattenOpenIaaSVirtualMachine(vm)

	// Lazy, single fetch of the VM-scoped device lists: a nil per-id
	// answer is only treated as a deletion after a second independent
	// evidence, because the OpenIaaS API answers 403 for unknown AND
	// forbidden ids alike and the client maps both to nil — a permission
	// hiccup must never silently shrink the state (#273).
	var listedDiskIDs map[string]bool
	confirmDiskGone := func(id string) (bool, error) {
		if listedDiskIDs == nil {
			// ListStrict: an access-denied listing must fail closed, not
			// masquerade as an empty list (the regular List maps 403 to nil).
			diskList, err := c.Compute().OpenIaaS().VirtualDisk().ListStrict(ctx, &client.OpenIaaSVirtualDiskFilter{
				VirtualMachineID: d.Id(),
			})
			if err != nil {
				return false, err
			}
			listedDiskIDs = map[string]bool{}
			for _, disk := range diskList {
				listedDiskIDs[disk.ID] = true
			}
		}
		return deviceConfirmedGone(listedDiskIDs, id), nil
	}

	// Get the OS disks
	osDisks := []interface{}{}
	for _, osDisk := range d.Get("os_disk").([]interface{}) {
		if osDisk == nil {
			continue
		}
		osDiskId := osDisk.(map[string]interface{})["id"].(string)
		if osDiskId == "" {
			continue
		}
		disk, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, osDiskId)
		if err != nil {
			return diag.Errorf("failed to read os disk: %s", err)
		}
		switch classifyOSDiskOnRead(disk, d.Id()) {
		case osDiskReadKeep:
			osDisks = append(osDisks, helpers.FlattenOpenIaaSOSDiskData(disk, d.Id()))
		case osDiskReadDropGone:
			gone, err := confirmDiskGone(osDiskId)
			if err != nil {
				return diag.Errorf("failed to confirm the deletion of os disk %s: %s", osDiskId, err)
			}
			if !gone {
				return diag.Errorf("os disk %s could not be read but is still attached to virtual machine %s: refusing to drop it from the state (possible access restriction)", osDiskId, d.Id())
			}
			// Deletion confirmed by two independent reads: reflect reality
			// in the state instead of crashing on the stale entry (#234
			// class).
		case osDiskReadDropPlatformManaged:
			// Legacy state pollution (#255): pre-1.8.0 creations could
			// capture the cloud-init config drive as a managed os_disk,
			// leaving a permanent removal drift. The strict live evidence
			// (exact XO name AND read-only VBD on this VM) allows the
			// cleanup; ambiguous disks stay in the state and require the
			// documented manual remediation. No platform-side change is
			// ever made: this is a state-shape correction only.
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Platform-managed cloud-init config drive removed from os_disk",
				Detail: fmt.Sprintf("The disk %s (%q) is the XO cloud-init config drive, attached read-only by the platform at deploy time. It was captured in the Terraform state by a pre-1.8.0 creation and is now dropped from os_disk to stop the permanent removal drift. No platform-side change was made.",
					osDiskId, disk.Name),
			})
		}
	}
	vmData["os_disk"] = osDisks

	// Get the OS network adapters
	var listedAdapterIDs map[string]bool
	osNetworkAdapters := []interface{}{}
	for _, osNetworkAdapter := range d.Get("os_network_adapter").([]interface{}) {
		if osNetworkAdapter == nil {
			continue
		}
		osNetworkAdapterId := osNetworkAdapter.(map[string]interface{})["id"].(string)
		if osNetworkAdapterId == "" {
			continue
		}
		networkAdapter, err := c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, osNetworkAdapterId)
		if err != nil {
			return diag.Errorf("failed to read os network adapter: %s", err)
		}
		if networkAdapter == nil {
			if listedAdapterIDs == nil {
				// ListStrict: an access-denied listing must fail closed,
				// not masquerade as an empty list.
				adapterList, err := c.Compute().OpenIaaS().NetworkAdapter().ListStrict(ctx, &client.OpenIaaSNetworkAdapterFilter{
					VirtualMachineID: d.Id(),
				})
				if err != nil {
					return diag.Errorf("failed to confirm the deletion of os network adapter %s: %s", osNetworkAdapterId, err)
				}
				listedAdapterIDs = map[string]bool{}
				for _, adapter := range adapterList {
					listedAdapterIDs[adapter.ID] = true
				}
			}
			if !deviceConfirmedGone(listedAdapterIDs, osNetworkAdapterId) {
				return diag.Errorf("os network adapter %s could not be read but is still attached to virtual machine %s: refusing to drop it from the state (possible access restriction)", osNetworkAdapterId, d.Id())
			}
			// Deletion confirmed by two independent reads: reflect reality
			// instead of crashing on the stale entry (#234 class).
			continue
		}
		osNetworkAdapters = append(osNetworkAdapters, helpers.FlattenOpenIaaSOSNetworkAdapterData(networkAdapter))
	}
	vmData["os_network_adapter"] = osNetworkAdapters

	tags, err := c.Tag().Resource().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to get tags: %s", err)
	}

	tagsMap := make(map[string]interface{})
	for _, tag := range tags {
		tagsMap[tag.Key] = tag.Value
	}
	d.Set("tags", tagsMap)

	// Get the SLA policies
	slaPolicies, err := c.Backup().OpenIaaS().Policy().List(ctx, &client.BackupOpenIaasPolicyFilter{
		VirtualMachineId: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to get sla policies: %s", err)
	}

	slaPoliciesIds := []string{}
	for _, slaPolicy := range slaPolicies {
		slaPoliciesIds = append(slaPoliciesIds, slaPolicy.ID)
	}

	// Get the replication information
	replicationPolicyId := ""
	replicationPolicy, err := c.Compute().OpenIaaS().Replication().Policy().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to get replication policy for virtual machine: %s", err)
	}
	if replicationPolicy != nil {
		replicationPolicyId = replicationPolicy.ID
	}

	vmData["backup_sla_policies"] = slaPoliciesIds
	vmData["replication_policy_id"] = replicationPolicyId

	// Set the data in the state
	for k, v := range vmData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func openIaasVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Associate a replication policy if provided
	if d.HasChange("replication_policy_id") {
		oldPolicyId, newPolicyId := d.GetChange("replication_policy_id")

		// Dissociate old policy if it exists
		if oldPolicyId.(string) != "" {
			activityId, err := c.Compute().OpenIaaS().Replication().Policy().VirtualMachine().Dissociate(ctx, d.Id())
			if err != nil {
				return diag.Errorf("failed to dissociate replication policy from virtual machine: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to dissociate replication policy from virtual machine: %s", err)
			}
		}

		// Associate new policy if provided
		if newPolicyId.(string) != "" {
			activityId, err := c.Compute().OpenIaaS().Replication().Policy().VirtualMachine().Associate(ctx, d.Id(), &client.AssociateReplicationPolicyToVirtualMachineRequest{
				ConfigurationID: newPolicyId.(string),
			})
			if err != nil {
				return diag.Errorf("failed to associate replication policy to virtual machine: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to associate replication policy to virtual machine: %s", err)
			}
		}
	}

	// PATCH only the properties that actually diverge from the live API
	// state (#267): during the create→update chaining every HasChange is
	// true while the deploy already applied the configuration, and values
	// merged through Computed are not write intent (#246 class).
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual machine state: %s", err)
	}
	if vm == nil {
		return diag.Errorf("failed to find virtual machine: %s", d.Id())
	}

	patch, changed, needsReboot := buildOpenIaasVMPropertiesPatch(vm, openIaasVMDesiredPropertiesFromResourceData(d))
	wasRunning := false

	if changed {
		// The cpu/memory/cores divergences require a power cycle first
		if needsReboot && vm.PowerState == "Running" {
			wasRunning = true

			if d.Get("allow_vm_restart").(bool) {
				// Power off the VM
				activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
					PowerState: "off",
					Force:      !vm.PVDrivers.Detected,
				})
				if err == nil {
					_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
				}
				if err != nil {
					// The power-off outcome is uncertain: try to restore
					// (the helper reads the live state first).
					if restoreErr := powerOnBestEffort(ctx, c, d.Id(), vm.Host.ID); restoreErr != nil {
						return diag.Errorf("failed to power off virtual machine: %s (WARNING: its power state could not be restored: %s)", err, restoreErr)
					}
					return diag.Errorf("failed to power off virtual machine: %s (the virtual machine is running)", err)
				}
			} else {
				return diag.Errorf("The virtual machine %s (%s) needs to be powered off to apply changes to cpu, memory or num_cores_per_socket. Please set allow_vm_restart to true to allow the provider to power off and on the VM if necessary.", vm.Name, d.Id())
			}
		}

		// Update the VM properties
		activityId, err := c.Compute().OpenIaaS().VirtualMachine().Update(ctx, d.Id(), patch)
		if err == nil {
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		}
		if err != nil {
			// The power-off was provider-initiated: never leave the VM
			// halted because the PATCH failed (FF-2).
			if wasRunning {
				if restoreErr := powerOnBestEffort(ctx, c, d.Id(), vm.Host.ID); restoreErr != nil {
					return diag.Errorf("failed to update virtual machine: %s (WARNING: the virtual machine was powered off for this update and could not be powered back on: %s)", err, restoreErr)
				}
				return diag.Errorf("failed to update virtual machine: %s (the virtual machine was powered back on)", err)
			}
			return diag.Errorf("failed to update virtual machine: %s", err)
		}
	}

	// If VM was running before and we powered it off, power it back on
	if needsReboot && wasRunning {
		activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
			PowerState: "on",
			Force:      false,
		})
		if err != nil {
			return diag.Errorf("failed to power on virtual machine: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power on virtual machine: %s", err)
		}

		// Wait for PV drivers to be available
		timeout := time.Duration(d.Get("wait_for_drivers_timeout").(int)) * time.Second
		if timeout > 0 {
			_, err = c.Compute().OpenIaaS().VirtualMachine().WaitForDrivers(ctx, d.Id(), timeout, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to get PV drivers on virtual machine after power on, %s", err)
			}
		}
	}

	if d.HasChange("mount_iso") {
		old, new := d.GetChange("mount_iso")

		if old != "" {
			activityId, err := c.Compute().OpenIaaS().VirtualMachine().UnmountISO(ctx, d.Id())
			if err != nil {
				return diag.Errorf("failed to unmount DVD drive: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to unmount DVD drive, %s", err)
			}
		}

		if new != "" {
			virtualDiskId := d.Get("mount_iso").(string)
			activityId, err := c.Compute().OpenIaaS().VirtualMachine().MountISO(ctx, d.Id(), virtualDiskId)
			if err != nil {
				return diag.Errorf("failed to mount DVD drive: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to mount DVD drive, %s", err)
			}
		}
	}

	if d.HasChange("boot_order") {
		bootOrder := d.Get("boot_order").([]interface{})
		bootOrderStr := make([]string, len(bootOrder))
		for i, v := range bootOrder {
			bootOrderStr[i] = v.(string)
		}
		activityId, err := c.Compute().OpenIaaS().VirtualMachine().UpdateBootOrder(ctx, d.Id(), bootOrderStr)
		if err != nil {
			return diag.Errorf("failed to update boot order: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update boot order, %s", err)
		}
	}

	// Handle os_disk and os_network_adapter updates
	// We don't handle adding/removing disks/vifs here, only updating existing ones
	// New disks should be added via a separate resource
	disks := []map[string]interface{}{}
	networkAdapters := []map[string]interface{}{}

	if d.HasChange("os_disk") {
		for _, disk := range d.Get("os_disk").([]interface{}) {
			if disk == nil {
				continue
			}
			disks = append(disks, disk.(map[string]interface{}))
		}
	}

	if d.HasChange("os_network_adapter") {
		for _, networkAdapter := range d.Get("os_network_adapter").([]interface{}) {
			if networkAdapter == nil {
				continue
			}
			networkAdapters = append(networkAdapters, networkAdapter.(map[string]interface{}))
		}
	}

	if diags := handleUpdateOSDevices(ctx, c, d, disks, networkAdapters); diags != nil {
		return diags
	}

	if d.HasChange("power_state") {
		powerState := d.Get("power_state").(string)
		// Avoid trying to power off a halted VM
		if powerState == "on" || !d.IsNewResource() {
			activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
				HostId:                  d.Get("host_id").(string),
				PowerState:              powerState,
				Force:                   false,
				BypassMacAddressesCheck: false,
				BypassBlockedOperation:  false,
				ForceShutdownDelay:      0,
			})
			if err != nil {
				return diag.Errorf("failed to power %s virtual machine: %s", powerState, err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to power %s virtual machine, %s", powerState, err)
			}
			// Avoid trying to wait for the drivers on a halter virtual machine
			if powerState != "off" {
				// We have to wait for the PV drivers to be detected, otherwise, operations like creating new network adapters will fail.
				timeout := time.Duration(d.Get("wait_for_drivers_timeout").(int)) * time.Second
				if timeout > 0 {
					_, err = c.Compute().OpenIaaS().VirtualMachine().WaitForDrivers(ctx, d.Id(), timeout, getWaiterOptions(ctx))
					if err != nil {
						return diag.Errorf("failed to get PV drivers on virtual machine, %s", err)
					}
				}
			}
		}
	}

	if diags := updateTags(ctx, c, d, d.Id(), "iaas_opensource_virtual_machine", "iaas_opensource"); diags != nil {
		return diags
	}

	return openIaasVirtualMachineRead(ctx, d, meta)
}

func openIaasVirtualMachineDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete virtual machine: %s", err)
	}
	if _, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete virtual machine, %s", err)
	}
	return nil
}

// osDiskPendingChanges describes the operations actually required to bring a
// disk in line with the desired configuration, based on the LIVE API state.
type osDiskPendingChanges struct {
	update   bool // size and/or name differ
	relocate bool // storage repository differs
}

// diskPendingChanges compares the desired os_disk block against the actual
// disk returned by the API. A nil actual disk conservatively requests both
// operations.
func diskPendingChanges(desired map[string]interface{}, actual *client.OpenIaaSVirtualDisk) osDiskPendingChanges {
	if actual == nil {
		return osDiskPendingChanges{update: true, relocate: true}
	}
	changes := osDiskPendingChanges{}
	if size, ok := desired["size"].(int); ok && size != actual.Size {
		changes.update = true
	}
	if name, ok := desired["name"].(string); ok && name != "" && name != actual.Name {
		changes.update = true
	}
	if srID, ok := desired["storage_repository_id"].(string); ok && srID != "" && srID != actual.StorageRepository.ID {
		changes.relocate = true
	}
	return changes
}

// osDiskReadAction is the decision of classifyOSDiskOnRead for one os_disk
// entry of the state during a refresh.
type osDiskReadAction int

const (
	// osDiskReadKeep: the disk exists and is user-managed — keep it.
	osDiskReadKeep osDiskReadAction = iota
	// osDiskReadDropGone: the disk no longer exists platform-side — drop
	// the stale entry instead of crashing on the nil API answer.
	osDiskReadDropGone
	// osDiskReadDropPlatformManaged: the disk is the XO cloud-init config
	// drive (strict live evidence: exact name AND read-only VBD on this
	// VM) captured by a pre-1.8.0 creation — drop it with a warning.
	osDiskReadDropPlatformManaged
)

// classifyOSDiskOnRead decides what the refresh does with an os_disk entry
// of the state, based exclusively on the live API answer. Ambiguity is
// conservative: a disk that merely shares the XO config drive name (writable,
// read-only on another VM, or without a VBD on this VM) stays managed.
func classifyOSDiskOnRead(disk *client.OpenIaaSVirtualDisk, virtualMachineID string) osDiskReadAction {
	if disk == nil {
		return osDiskReadDropGone
	}
	if helpers.IsPlatformManagedDisk(disk, virtualMachineID) {
		return osDiskReadDropPlatformManaged
	}
	return osDiskReadKeep
}

// powerOnBestEffort powers the VM back on after a failed operation that
// required a provider-initiated power-off: leaving the VM halted because
// the operation failed would turn a failed update into an outage (FF-2).
// It reads the live power state first, so it is safe to call when the
// power-off outcome is uncertain: an already-running VM is a success and
// no second power-on is sent.
func powerOnBestEffort(ctx context.Context, c *client.Client, vmID string, hostID string) error {
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, vmID)
	if err != nil {
		return fmt.Errorf("could not read the power state: %s", err)
	}
	if vm == nil {
		return fmt.Errorf("virtual machine %s not found", vmID)
	}
	if vm.PowerState == "Running" {
		return nil
	}
	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, vmID, &client.UpdateOpenIaasVirtualMachinePowerRequest{
		PowerState: "on",
		HostId:     hostID,
	})
	if err != nil {
		return err
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	return err
}

// deviceConfirmedGone reports whether a device id absent from the per-id
// read is also absent from the VM-scoped listing — the second independent
// evidence required before dropping a state entry (#273).
func deviceConfirmedGone(listedIDs map[string]bool, id string) bool {
	return !listedIDs[id]
}

// openIaasVMDesiredProperties carries the desired VM properties from the
// plan. The Optional+Computed attributes (SecureBoot, NumCoresPerSocket,
// BootFirmware) hold raw-config evidence: the merged plan value is not
// write intent, so nil means "not explicitly configured, never push" —
// even when the state-merged value diverges from the live one (a stale
// state must not overwrite a live setting, #246 class / FF-2). AutoPowerOn
// is a plain Optional boolean: its plan value is authoritative (absent
// means false by the Terraform contract) and it is always set.
type openIaasVMDesiredProperties struct {
	Name              string
	CPU               int
	Memory            int
	HighAvailability  string
	NumCoresPerSocket *int
	BootFirmware      *string
	SecureBoot        *bool
	AutoPowerOn       *bool
}

// openIaasVMDesiredPropertiesFromResourceData extracts the desired VM
// properties and the raw-config evidence for the Optional+Computed
// attributes.
func openIaasVMDesiredPropertiesFromResourceData(d *schema.ResourceData) openIaasVMDesiredProperties {
	desired := openIaasVMDesiredProperties{
		Name:             d.Get("name").(string),
		CPU:              d.Get("cpu").(int),
		Memory:           d.Get("memory").(int),
		HighAvailability: d.Get("high_availability").(string),
	}
	autoPowerOn := d.Get("auto_power_on").(bool)
	desired.AutoPowerOn = &autoPowerOn
	if raw := d.GetRawConfig(); !raw.IsNull() && raw.IsKnown() {
		if v := raw.GetAttr("secure_boot"); !v.IsNull() && v.IsKnown() {
			secureBoot := v.True()
			desired.SecureBoot = &secureBoot
		}
		if v := raw.GetAttr("num_cores_per_socket"); !v.IsNull() && v.IsKnown() {
			cores, _ := v.AsBigFloat().Int64()
			coresInt := int(cores)
			desired.NumCoresPerSocket = &coresInt
		}
		if v := raw.GetAttr("boot_firmware"); !v.IsNull() && v.IsKnown() {
			firmware := v.AsString()
			desired.BootFirmware = &firmware
		}
	}
	return desired
}

// buildOpenIaasVMPropertiesPatch compares the desired properties against
// the live API state and returns the PATCH limited to the real
// divergences. changed reports whether the PATCH carries anything;
// needsReboot reports whether one of the included fields (cpu, memory,
// num_cores_per_socket) requires a power cycle. Zero/empty desired values
// never overwrite live values: a value merged through Computed is not
// write intent (#246 class, #267).
func buildOpenIaasVMPropertiesPatch(live *client.OpenIaaSVirtualMachine, desired openIaasVMDesiredProperties) (*client.UpdateOpenIaasVirtualMachineRequest, bool, bool) {
	req := &client.UpdateOpenIaasVirtualMachineRequest{}
	changed := false
	needsReboot := false

	if desired.Name != "" && desired.Name != live.Name {
		req.Name = desired.Name
		changed = true
	}
	if desired.CPU != 0 && desired.CPU != live.CPU {
		req.CPU = desired.CPU
		changed = true
		needsReboot = true
	}
	if desired.NumCoresPerSocket != nil && *desired.NumCoresPerSocket != 0 && *desired.NumCoresPerSocket != live.NumCoresPerSocket {
		req.NumCoresPerSocket = *desired.NumCoresPerSocket
		changed = true
		needsReboot = true
	}
	if desired.Memory != 0 && desired.Memory != live.Memory {
		req.Memory = desired.Memory
		changed = true
		needsReboot = true
	}
	if desired.BootFirmware != nil && *desired.BootFirmware != "" && *desired.BootFirmware != live.BootFirmware {
		req.BootFirmware = *desired.BootFirmware
		changed = true
	}
	if desired.HighAvailability != "" && desired.HighAvailability != live.HighAvailability {
		req.HighAvailability = desired.HighAvailability
		changed = true
	}
	if desired.SecureBoot != nil && *desired.SecureBoot != live.SecureBoot {
		req.SecureBoot = desired.SecureBoot
		changed = true
	}
	if desired.AutoPowerOn != nil && *desired.AutoPowerOn != live.AutoPowerOn {
		req.AutoPowerOn = desired.AutoPowerOn
		changed = true
	}

	return req, changed, needsReboot
}

// adapterNeedsUpdate compares the desired os_network_adapter block against
// the actual adapter returned by the API. An empty desired value never
// triggers an update (an unset MAC must not be pushed). tx_checksumming is
// Optional+Computed and the post-create state merge does not retain
// explicit false booleans, so the desired map cannot be trusted for it:
// the divergence is evaluated against the explicitly configured raw value
// (txWant, nil when the block does not configure it).
func adapterNeedsUpdate(desired map[string]interface{}, actual *client.OpenIaaSNetworkAdapter, txWant *bool) bool {
	if actual == nil {
		return true
	}
	if networkID, ok := desired["network_id"].(string); ok && networkID != "" && networkID != actual.Network.ID {
		return true
	}
	if mac, ok := desired["mac_address"].(string); ok && mac != "" && !strings.EqualFold(mac, actual.MacAddress) {
		return true
	}
	if txWant != nil && *txWant != actual.TxChecksumming {
		return true
	}
	return false
}

// osAdapterTxConfigured returns, keyed by adapter id, the tx_checksumming
// value explicitly set by the os_network_adapter block at the same index in
// the raw user configuration (raw is d.GetRawConfig()); absent ids mean the
// block does not configure it. The raw value is authoritative: the merged
// desired map seeds tx_checksumming from the live adapter (Computed) and
// the state merge swallows explicit false values, which would either push
// an unrequested VIF PATCH (#246) or skip a requested one on first apply.
// The raw config list is aligned by index with the unfiltered
// d.Get("os_network_adapter") list. An unknown raw value cannot occur
// during apply and has no concrete value to push: it stays unconfigured
// (fail-safe, no PATCH).
func osAdapterTxConfigured(raw cty.Value, adapters []interface{}) map[string]*bool {
	configured := map[string]*bool{}
	if raw.IsNull() || !raw.IsKnown() {
		return configured
	}
	rawAdapters := raw.GetAttr("os_network_adapter")
	if rawAdapters.IsNull() || !rawAdapters.IsKnown() {
		return configured
	}
	rawList := rawAdapters.AsValueSlice()
	for i, adapter := range adapters {
		if i >= len(rawList) {
			break
		}
		desired, ok := adapter.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := desired["id"].(string)
		if id == "" {
			continue
		}
		if v := rawList[i].GetAttr("tx_checksumming"); !v.IsNull() && v.IsKnown() {
			tx := v.True()
			configured[id] = &tx
		}
	}
	return configured
}

func handleUpdateOSDevices(ctx context.Context, c *client.Client, d *schema.ResourceData, disks []map[string]interface{}, networkAdapters []map[string]interface{}) diag.Diagnostics {
	// Nothing to reconcile: do not make an unrelated update (tags, power
	// state, boot order…) depend on the disk/adapter listing endpoints.
	if len(disks) == 0 && len(networkAdapters) == 0 {
		return nil
	}

	needsReboot := false

	// Read the current state of the VM to check its state and disk connections
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual machine: %s", err)
	} else if vm == nil {
		return diag.Errorf("failed to find virtual machine: %s", d.Id())
	}

	// Compare the desired configuration against the LIVE API state instead of
	// relying on d.HasChange alone: during the create→update chaining every
	// HasChange is true, while the marketplace deploy has already applied the
	// network mapping (networkData). Pushing unconditional VIF updates turned
	// a platform-side incident on that single operation into a full
	// provisioning failure for otherwise healthy VMs (issue #246).
	actualDisks := map[string]*client.OpenIaaSVirtualDisk{}
	diskList, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, &client.OpenIaaSVirtualDiskFilter{
		VirtualMachineID: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to list virtual disks: %s", err)
	}
	for _, disk := range diskList {
		actualDisks[disk.ID] = disk
	}
	actualAdapters := map[string]*client.OpenIaaSNetworkAdapter{}
	adapterList, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, &client.OpenIaaSNetworkAdapterFilter{
		VirtualMachineID: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to list network adapters: %s", err)
	}
	for _, adapter := range adapterList {
		actualAdapters[adapter.ID] = adapter
	}

	// Indexes the raw configuration before the nil-filtered adapter slice is
	// walked: the raw config list is aligned with the unfiltered d.Get list.
	txConfigured := osAdapterTxConfigured(d.GetRawConfig(), d.Get("os_network_adapter").([]interface{}))

	pendingDisks := map[string]osDiskPendingChanges{}
	for _, disk := range disks {
		id, ok := disk["id"].(string)
		if !ok || id == "" {
			return diag.Errorf("os_disk without id in the state of virtual machine %s: cannot reconcile (partial or corrupted state)", d.Id())
		}
		actual, found := actualDisks[id]
		if !found {
			// Never act (let alone power off the VM) on a device the API
			// does not know about: surface the divergence instead.
			return diag.Errorf("os_disk %s is in the Terraform state but not returned by the API for virtual machine %s: refresh the state before updating", id, d.Id())
		}
		if changes := diskPendingChanges(disk, actual); changes.update || changes.relocate {
			pendingDisks[id] = changes
			if changes.update && vm.PowerState == "Running" {
				needsReboot = true
			}
		}
	}

	pendingAdapters := map[string]bool{}
	for _, networkAdapter := range networkAdapters {
		id, ok := networkAdapter["id"].(string)
		if !ok || id == "" {
			return diag.Errorf("os_network_adapter without id in the state of virtual machine %s: cannot reconcile (partial or corrupted state)", d.Id())
		}
		actual, found := actualAdapters[id]
		if !found {
			return diag.Errorf("os_network_adapter %s is in the Terraform state but not returned by the API for virtual machine %s: refresh the state before updating", id, d.Id())
		}
		if adapterNeedsUpdate(networkAdapter, actual, txConfigured[id]) {
			pendingAdapters[id] = true
			mac, _ := networkAdapter["mac_address"].(string)
			if mac != "" && !strings.EqualFold(mac, actual.MacAddress) && vm.PowerState == "Running" {
				needsReboot = true
			}
		}
	}

	if len(pendingDisks) == 0 && len(pendingAdapters) == 0 {
		return nil
	}

	// If a reboot is necessary, check that the user has allowed the provider to restart the VM
	if needsReboot && !d.Get("allow_vm_restart").(bool) {
		return diag.Errorf("The virtual machine %s (%s) needs to be powered off to apply changes to the os_disks/os_network_adapters. Please set allow_vm_restart to true to allow the provider to power off and on the VM if necessary.", vm.Name, d.Id())
	}

	poweredOff := false
	if needsReboot && d.Get("allow_vm_restart").(bool) {
		// Power off the VM
		activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
			PowerState: "off",
			Force:      !vm.PVDrivers.Detected, // If PV drivers are not detected, force shutdown to avoid communication issues with the VM, otherwise do a soft shutdown.
		})
		if err == nil {
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		}
		if err != nil {
			// The power-off outcome is uncertain: try to restore (the
			// helper reads the live state first).
			if restoreErr := powerOnBestEffort(ctx, c, d.Id(), vm.Host.ID); restoreErr != nil {
				return diag.Errorf("failed to power off virtual machine: %s (WARNING: its power state could not be restored: %s)", err, restoreErr)
			}
			return diag.Errorf("failed to power off virtual machine: %s (the virtual machine is running)", err)
		}
		poweredOff = true
	}

	// Apply the divergences; on any failure after a provider-initiated
	// power-off, power the VM back on before surfacing the error (FF-2).
	applyDiags := func() diag.Diagnostics {
		// Apply modifications to the disks that actually diverge from the API
		for _, disk := range disks {
			id, _ := disk["id"].(string)
			changes, pending := pendingDisks[id]
			if !pending {
				continue
			}
			if diags := osDiskUpdate(ctx, c, disk, changes); diags != nil {
				return diags
			}
		}
		return nil
	}()
	if applyDiags != nil {
		if poweredOff {
			if restoreErr := powerOnBestEffort(ctx, c, d.Id(), vm.Host.ID); restoreErr != nil {
				return append(applyDiags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Virtual machine left powered off",
					Detail:   fmt.Sprintf("The virtual machine %s was powered off for this update and could not be powered back on: %s", d.Id(), restoreErr),
				})
			}
			return append(applyDiags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Virtual machine powered back on after a failed update",
			})
		}
		return applyDiags
	}

	// Apply modifications to the network adapters that actually diverge
	adapterDiags := func() diag.Diagnostics {
		for _, networkAdapter := range networkAdapters {
			id, _ := networkAdapter["id"].(string)
			if !pendingAdapters[id] {
				continue
			}
			if diags := osNetworkAdapterUpdate(ctx, c, networkAdapter, actualAdapters[id], txConfigured[id]); diags != nil {
				return diags
			}
		}
		return nil
	}()
	if adapterDiags != nil {
		if poweredOff {
			if restoreErr := powerOnBestEffort(ctx, c, d.Id(), vm.Host.ID); restoreErr != nil {
				return append(adapterDiags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "Virtual machine left powered off",
					Detail:   fmt.Sprintf("The virtual machine %s was powered off for this update and could not be powered back on: %s", d.Id(), restoreErr),
				})
			}
			return append(adapterDiags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Virtual machine powered back on after a failed update",
			})
		}
		return adapterDiags
	}

	// Power on the VM if it was powered off for the update
	if needsReboot {
		activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
			PowerState: "on",
			HostId:     d.Get("host_id").(string),
		})
		if err != nil {
			return diag.Errorf("failed to power on virtual machine: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to power on virtual machine: %s", err)
		}
	}

	return nil
}

func osDiskUpdate(ctx context.Context, c *client.Client, disk map[string]interface{}, changes osDiskPendingChanges) diag.Diagnostics {
	// Update the disk if necessary
	if changes.update {
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Update(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskUpdateRequest{
			Size: disk["size"].(int),
			Name: disk["name"].(string),
		})
		if err != nil {
			return diag.Errorf("failed to update virtual disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update virtual disk, %s", err)
		}
	}

	// Handle the disk relocation if necessary
	if changes.relocate {
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Relocate(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskRelocateRequest{
			StorageRepositoryID: disk["storage_repository_id"].(string),
		})
		if err != nil {
			return diag.Errorf("failed to relocate virtual disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to relocate virtual disk, %s", err)
		}
	}

	return nil
}

// buildOpenIaasVIFPatch returns the PATCH limited to the fields that
// actually diverge from the live adapter, or nil when the adapter is
// already converged: re-sending the current networkId/mac is rejected
// platform-side as a VPC Static IP self-conflict (#246). tx_checksumming
// is only sent when explicitly configured in the block, using the raw
// config value (the merged map swallows explicit false on first apply).
func buildOpenIaasVIFPatch(networkAdapter map[string]interface{}, actual *client.OpenIaaSNetworkAdapter, txWant *bool) *client.UpdateOpenIaasNetworkAdapterRequest {
	req := &client.UpdateOpenIaasNetworkAdapterRequest{}
	if networkID, _ := networkAdapter["network_id"].(string); networkID != "" && networkID != actual.Network.ID {
		req.NetworkID = networkID
	}
	if mac, _ := networkAdapter["mac_address"].(string); mac != "" && !strings.EqualFold(mac, actual.MacAddress) {
		req.MAC = mac
	}
	if txWant != nil && *txWant != actual.TxChecksumming {
		req.TxChecksumming = txWant
	}
	if req.NetworkID == "" && req.MAC == "" && req.TxChecksumming == nil {
		return nil
	}
	return req
}

func osNetworkAdapterUpdate(ctx context.Context, c *client.Client, networkAdapter map[string]interface{}, actual *client.OpenIaaSNetworkAdapter, txWant *bool) diag.Diagnostics {
	if buildOpenIaasVIFPatch(networkAdapter, actual, txWant) == nil {
		return nil
	}
	adapterID := networkAdapter["id"].(string)
	// Bounded retry on transient platform failures, rebuilding the payload
	// against a freshly read live adapter before every attempt (#251).
	err := runVIFUpdateWithRetry(ctx, adapterID, clientVIFUpdateFuncs(c, adapterID, getWaiterOptions(ctx)), func(actual *client.OpenIaaSNetworkAdapter) *client.UpdateOpenIaasNetworkAdapterRequest {
		return buildOpenIaasVIFPatch(networkAdapter, actual, txWant)
	})
	if err != nil {
		return diag.Errorf("failed to update os network adapter: %s", err)
	}

	return nil
}
