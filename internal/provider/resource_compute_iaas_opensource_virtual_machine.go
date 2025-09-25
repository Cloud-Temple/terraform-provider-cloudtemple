package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
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
				Type:         schema.TypeString,
				Description:  "The template identifier.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"cpu": {
				Type:        schema.TypeInt,
				Description: "The number of virtual CPUs. Note: Changing this value for a running VM will cause it to be powered off and back on.",
				Required:    true,
			},
			"num_cores_per_socket": {
				Type:        schema.TypeInt,
				Description: "The number of cores per socket. Note: Changing this value for a running VM will cause it to be powered off and back on.",
				Optional:    true,
				Computed:    true,
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
				Type:         schema.TypeString,
				Description:  "The boot firmware to use. Available values are 'bios' and 'uefi'.",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"bios", "uefi"}, false),
			},
			"auto_power_on": {
				Type:        schema.TypeBool,
				Description: "Whether to automatically start the virtual machine when the host boots.",
				Optional:    true,
			},
			"high_availability": {
				Type:         schema.TypeString,
				Description:  "HA mode to enable on the virtual machine.",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"best-effort", "restart"}, false),
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
							Description: "Whether the network adapter is attached.",
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

func openIaasVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Cloud-Init to configure the virtual machine
	var cloudInit client.CloudInit
	cloudInitRaw, ok := d.Get("cloud_init").(map[string]interface{})
	if ok && cloudInitRaw != nil && len(cloudInitRaw) > 0 {
		cloudConfig, ok := cloudInitRaw["cloud_config"].(string)
		if cloudConfig != "" && ok {
			cloudInit.CloudConfig = cloudConfig
		}
		networkConfig, ok := cloudInitRaw["network_config"].(string)
		if networkConfig != "" && ok {
			cloudInit.NetworkConfig = networkConfig
		}
	}

	template, err := c.Compute().OpenIaaS().Template().Read(ctx, d.Get("template_id").(string))
	if err != nil {
		return diag.Errorf("Could not read the template : %s", err)
	}
	if template == nil {
		return diag.Errorf("Could not find template with id : %s", d.Get("template_id").(string))
	}

	templateNetworkAdapters := make([]client.OSNetworkAdapter, len(template.NetworkAdapters))
	for i, networkAdapter := range template.NetworkAdapters {
		templateNetworkAdapters[i] = client.OSNetworkAdapter{
			NetworkID: networkAdapter.Network.ID,
			MAC:       networkAdapter.MacAddress,
		}
	}

	// Create virtual machine itself
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

	osNetworkAdapters := helpers.UpdateNestedMapItems(d, helpers.FlattenOpenIaaSOSNetworkAdaptersData(networkAdapters), "os_network_adapter")
	if err := d.Set("os_network_adapter", osNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	// Assign SLA policies to the virtual machine
	slaPolicies := []string{}
	for _, policy := range d.Get("backup_sla_policies").(*schema.Set).List() {
		slaPolicies = append(slaPolicies, policy.(string))
	}
	activityId, err = c.Backup().OpenIaaS().Policy().Assign(ctx, &client.BackupOpenIaasAssignPolicyRequest{
		VirtualMachineId: d.Id(),
		PolicyIds:        slaPolicies,
	})
	if err != nil {
		return diag.Errorf("failed to assign policies to virtual machine, %s", err)
	}

	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to assign policies to virtual machine, %s", err)
	}

	return openIaasVirtualMachineUpdate(ctx, d, meta)
}

func openIaasVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	// Récupérer la machine virtuelle par son ID
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("the virtual machine could not be read: %s", err)
	}
	if vm == nil {
		d.SetId("") // La VM n'existe plus, marquer la ressource comme supprimée
		return nil
	}

	// Normaliser le power state pour qu'il soit cohérent avec l'entrée
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

	disks, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, &client.OpenIaaSVirtualDiskFilter{
		VirtualMachineID: d.Id(),
	})
	if err != nil {
		return diag.Errorf("failed to read os_disks, %s", err)
	}
	if disks == nil {
		return diag.Errorf("could not list disks of virtual machine, %s", err)
	}

	// Overwrite with the desired config
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

	osNetworkAdapters := helpers.UpdateNestedMapItems(d, helpers.FlattenOpenIaaSOSNetworkAdaptersData(networkAdapters), "os_network_adapter")
	if err := d.Set("os_network_adapter", osNetworkAdapters); err != nil {
		return diag.FromErr(err)
	}

	tags, err := c.Tag().Resource().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to get tags: %s", err)
	}

	tagsMap := make(map[string]interface{})
	for _, tag := range tags {
		tagsMap[tag.Key] = tag.Value
	}
	d.Set("tags", tagsMap)

	// Récupérer les SLA policies
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

	// Mapper les données en utilisant la fonction helper
	vmData := helpers.FlattenOpenIaaSVirtualMachine(vm)
	vmData["backup_sla_policies"] = slaPoliciesIds

	// Définir les données dans le state
	for k, v := range vmData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func openIaasVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Check if hardware-related properties have changed
	needsReboot := d.HasChange("cpu") || d.HasChange("memory") || d.HasChange("num_cores_per_socket")
	wasRunning := false

	// If hardware-related properties have changed, we need to power off the VM first
	if needsReboot {
		// Get current VM state to check if it's running
		vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return diag.Errorf("failed to read virtual machine state: %s", err)
		}

		// Only power off if the VM is running
		if vm.PowerState == "Running" {
			wasRunning = true

			// Power off the VM
			activityId, err := c.Compute().OpenIaaS().VirtualMachine().Power(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachinePowerRequest{
				PowerState: "off",
				Force:      true,
			})
			if err != nil {
				return diag.Errorf("failed to power off virtual machine: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to power off virtual machine: %s", err)
			}
		}
	}

	// Update the VM properties
	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Update(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachineRequest{
		Name:              d.Get("name").(string),
		CPU:               d.Get("cpu").(int),
		NumCoresPerSocket: d.Get("num_cores_per_socket").(int),
		Memory:            d.Get("memory").(int),
		BootFirmware:      d.Get("boot_firmware").(string),
		SecureBoot:        d.Get("secure_boot").(bool),
		AutoPowerOn:       d.Get("auto_power_on").(bool),
		HighAvailability:  d.Get("high_availability").(string),
	})
	if err != nil {
		return diag.Errorf("failed to update virtual machine: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update virtual machine, %s", err)
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

		// Wait for tools to be available
		_, err = c.Compute().OpenIaaS().VirtualMachine().WaitForTools(ctx, d.Id(), getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to get tools on virtual machine after power on, %s", err)
		}
	}

	// if d.HasChange("os_disk") {
	// 	for i, osDisk := range d.Get("os_disk").([]interface{}) {
	// 		if osDisk == nil {
	// 			continue
	// 		}
	// 		disk := osDisk.(map[string]interface{})
	// 		if d.HasChange(fmt.Sprintf("os_disk.%d.size", i)) || d.HasChange(fmt.Sprintf("os_disk.%d.name", i)) {
	// 			// TODO : Disconnect disk before updating (and specify it in documentation)
	// 			diskState, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, disk["id"].(string))
	// 			if err != nil {
	// 				return diag.Errorf("failed to read virtual disk state: %s", err)
	// 			}

	// 			vbd := Find(diskState.VirtualMachines, func(virtualMachine client.OpenIaaSVirtualDiskConnection) bool {
	// 				return virtualMachine.ID == d.Id()
	// 			})
	// 			if vbd.Connected {
	// 				// wasConnected := true
	// 				activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, &client.OpenIaaSVirtualDiskConnectionRequest{
	// 					ID:               disk["id"].(string),
	// 					VirtualMachineID: d.Id(),
	// 				})
	// 				if err != nil {
	// 					return diag.Errorf("failed to disconnect os disk: %s", err)
	// 				}
	// 				_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	// 				if err != nil {
	// 					return diag.Errorf("failed to disconnect os disk: %s", err)
	// 				}
	// 			}

	// 			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Update(ctx, &client.OpenIaaSVirtualDiskUpdateRequest{
	// 				ID:   disk["id"].(string),
	// 				Size: disk["size"].(int),
	// 				Name: disk["name"].(string),
	// 			})
	// 			if err != nil {
	// 				return diag.Errorf("failed to update virtual disk: %s", err)
	// 			}
	// 			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	// 			if err != nil {
	// 				return diag.Errorf("failed to update virtual disk, %s", err)
	// 			}
	// 		}

	// 		if d.HasChange(fmt.Sprintf("os_disk.%d.storage_repository_id", i)) {
	// 			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Relocate(ctx, &client.OpenIaaSVirtualDiskRelocateRequest{
	// 				ID:                  disk["id"].(string),
	// 				StorageRepositoryID: disk["storage_repository_id"].(string),
	// 			})
	// 			if err != nil {
	// 				return diag.Errorf("failed to update virtual disk: %s", err)
	// 			}
	// 			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	// 			if err != nil {
	// 				return diag.Errorf("failed to update virtual disk, %s", err)
	// 			}
	// 		}

	// 		// if d.HasChange(fmt.Sprintf("os_disk.%d.connected", i)) {

	// 		// }
	// 	}
	// }

	if d.HasChange("os_disk") {
		for i, osDisk := range d.Get("os_disk").([]interface{}) {
			if osDisk == nil {
				continue
			}
			disk := osDisk.(map[string]interface{})

			if diags := osDiskUpdate(ctx, c, d, i, disk); diags != nil {
				return diags
			}
		}
	}

	if d.HasChange("os_network_adapter") {
		for i, osNetworkAdapter := range d.Get("os_network_adapter").([]interface{}) {
			if osNetworkAdapter == nil {
				continue
			}
			disk := osNetworkAdapter.(map[string]interface{})

			if diags := osNetworkAdapterUpdate(ctx, c, d, i, disk); diags != nil {
				return diags
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
			// We have to wait for the tools to be mounted, otherwise, operations like creating new network adapters will fail. If tools are not found after 30 seconds, we continue anyway.
			_, err = c.Compute().OpenIaaS().VirtualMachine().WaitForTools(ctx, d.Id(), getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to get tools on virtual machine, %s", err)
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

// handleOSDiskUpdate gère la mise à jour d'un disque OS, y compris la reconnexion si nécessaire
func osDiskUpdate(ctx context.Context, c *client.Client, d *schema.ResourceData, i int, disk map[string]interface{}) diag.Diagnostics {
	// Vérifier si on doit modifier la taille ou le nom
	needsDisconnect := d.HasChange(fmt.Sprintf("os_disk.%d.size", i)) || d.HasChange(fmt.Sprintf("os_disk.%d.name", i))

	// Vérifier l'état actuel du disque
	diskState, err := c.Compute().OpenIaaS().VirtualDisk().Read(ctx, disk["id"].(string))
	if err != nil {
		return diag.Errorf("failed to read virtual disk state: %s", err)
	}

	// Trouver la connexion pour cette VM
	vbd := helpers.Find(diskState.VirtualMachines, func(virtualMachine client.OpenIaaSVirtualDiskConnection) bool {
		return virtualMachine.ID == d.Id()
	})

	// Sauvegarder l'état de connexion initial
	wasConnected := vbd.Connected

	// Vérifier si l'utilisateur a modifié la propriété "connected"
	userChangedConnected := d.HasChange(fmt.Sprintf("os_disk.%d.connected", i))

	// Déconnecter si nécessaire pour la mise à jour
	if needsDisconnect && vbd.Connected {
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskConnectionRequest{
			VirtualMachineID: d.Id(),
		})
		if err != nil {
			return diag.Errorf("failed to disconnect os disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to disconnect os disk: %s", err)
		}
	}

	// Mettre à jour le disque si nécessaire
	if needsDisconnect {
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

	// Gérer le déplacement du disque si nécessaire
	if d.HasChange(fmt.Sprintf("os_disk.%d.storage_repository_id", i)) {
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

	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual machine : %s", err)
	}
	if vm == nil {
		return diag.Errorf("failed to find virtual machine : %s", d.Id())
	}

	// Gérer l'état de connexion seulement si la VM est allumée (sinon, risque d'erreur)
	if userChangedConnected && vm.PowerState == "Running" {
		// Utiliser la valeur définie par l'utilisateur
		wantConnected := disk["connected"].(bool)

		// Si l'état actuel ne correspond pas à l'état souhaité
		if wantConnected && !vbd.Connected {
			// Connecter le disque
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Connect(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: d.Id(),
			})
			if err != nil {
				return diag.Errorf("failed to connect os disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to connect os disk: %s", err)
			}
		} else if !wantConnected && vbd.Connected {
			// Déconnecter le disque
			activityId, err := c.Compute().OpenIaaS().VirtualDisk().Disconnect(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskConnectionRequest{
				VirtualMachineID: d.Id(),
			})
			if err != nil {
				return diag.Errorf("failed to disconnect os disk: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to disconnect os disk: %s", err)
			}
		}
	} else if needsDisconnect && wasConnected {
		// Si l'utilisateur n'a pas modifié l'état de connexion et que le disque était connecté avant
		// et qu'on l'a déconnecté pour la mise à jour, on le reconnecte
		activityId, err := c.Compute().OpenIaaS().VirtualDisk().Connect(ctx, disk["id"].(string), &client.OpenIaaSVirtualDiskConnectionRequest{
			VirtualMachineID: d.Id(),
		})
		if err != nil {
			return diag.Errorf("failed to reconnect os disk: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to reconnect os disk: %s", err)
		}
	}

	return nil
}

func osNetworkAdapterUpdate(ctx context.Context, c *client.Client, d *schema.ResourceData, i int, networkAdapter map[string]interface{}) diag.Diagnostics {
	vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read virtual machine : %s", err)
	}
	if vm == nil {
		return diag.Errorf("failed to find virtual machine : %s", d.Id())
	}

	if d.HasChange(fmt.Sprintf("os_network_adapter.%d.attached", i)) && vm.PowerState == "Running" {
		if networkAdapter["attached"].(bool) {
			activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Connect(ctx, networkAdapter["id"].(string))
			if err != nil {
				return diag.Errorf("failed to connect os network adapter: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to connect os network adapter: %s", err)
			}
		} else {
			activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Disconnect(ctx, networkAdapter["id"].(string))
			if err != nil {
				return diag.Errorf("failed to disconnect os network adapter: %s", err)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
			if err != nil {
				return diag.Errorf("failed to disconnect os network adapter: %s", err)
			}
		}
	}

	if d.HasChange(fmt.Sprintf("os_network_adapter.%d.mac_address", i)) || d.HasChange(fmt.Sprintf("os_network_adapter.%d.network_id", i)) {
		activityId, err := c.Compute().OpenIaaS().NetworkAdapter().Update(ctx, networkAdapter["id"].(string), &client.UpdateOpenIaasNetworkAdapterRequest{
			NetworkID: networkAdapter["network_id"].(string),
			MAC:       networkAdapter["mac_address"].(string),
		})
		if err != nil {
			return diag.Errorf("failed to update os network adapter: %s", err)
		}
		_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
		if err != nil {
			return diag.Errorf("failed to update os network adapter: %s", err)
		}
	}

	return nil
}
