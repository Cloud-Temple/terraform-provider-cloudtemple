package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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
				Description: "The number of virtual CPUs.",
				Required:    true,
			},
			"num_cores_per_socket": {
				Type:        schema.TypeInt,
				Description: "The number of cores per socket.",
				Optional:    true,
			},
			"memory": {
				Type:        schema.TypeInt,
				Description: "The amount of memory in MB.",
				Required:    true,
			},
			"power_state": {
				Type:         schema.TypeString,
				Description:  "The desired power state of the virtual machine.",
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"on", "off"}, false),
			},
			"host_id": {
				Type:         schema.TypeString,
				Description:  "The host identifier.",
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"boot_order": {
				Type:        schema.TypeList,
				Description: "The boot order of the virtual machine.",
				Optional:    true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"Hard-Drive", "DVD-Drive", "Network"}, false),
				},
			},
			"secure_boot": {
				Type:        schema.TypeBool,
				Description: "Whether to enable secure boot.",
				Optional:    true,
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

			//Out
			"machine_manager": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dvd_drive": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"tools": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"operating_system_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"addresses": {
				Type:     schema.TypeList,
				Computed: true,
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
			"pool": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"host": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// TODO : Add Tags (needs tag module update)

func openIaasVirtualMachineCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Create virtual machine itself
	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Create(ctx, &client.CreateOpenIaasVirtualMachineRequest{
		Name:       d.Get("name").(string),
		TemplateID: d.Get("template_id").(string),
		CPU:        d.Get("cpu").(int),
		Memory:     d.Get("memory").(int),
	})
	if err != nil {
		return diag.Errorf("the virtual machine could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityConcernedItems(d, activity, "virtual_machine")
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
	reader := readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
		vm, err := client.Compute().OpenIaaS().VirtualMachine().Read(ctx, d.Id())
		if err != nil {
			return nil, err
		}
		if vm == nil {
			return nil, nil
		}

		// Normalize the power state so that we can use it as input
		switch vm.PowerState {
		case "Running":
			vm.PowerState = "on"
		case "Halted":
			vm.PowerState = "off"
		default:
			return nil, fmt.Errorf("unknown power state %q", vm.PowerState)
		}

		readTags(ctx, sw, client, d.Id())

		return vm, nil
	})

	return reader(ctx, d, meta)
}

func openIaasVirtualMachineUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().VirtualMachine().Update(ctx, d.Id(), &client.UpdateOpenIaasVirtualMachineRequest{
		Name:              d.Get("name").(string),
		CPU:               d.Get("cpu").(int),
		NumCoresPerSocket: d.Get("num_cores_per_socket").(int),
		Memory:            d.Get("memory").(int),
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
