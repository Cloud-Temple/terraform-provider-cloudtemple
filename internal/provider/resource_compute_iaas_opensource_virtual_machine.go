package provider

import (
	"context"
	"regexp"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
	// Create virtual machine itself
	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Create(ctx, &client.CreateOpenIaasVirtualMachineRequest{
		Name:       d.Get("name").(string),
		TemplateID: d.Get("template_id").(string),
		CPU:        d.Get("cpu").(int),
		Memory:     d.Get("memory").(int),
		CloudInit:  cloudInit,
	})
	if err != nil {
		return diag.Errorf("the virtual machine could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityState(d, activity)
	if err != nil {
		return diag.Errorf("failed to create virtual machine, %s", err)
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
