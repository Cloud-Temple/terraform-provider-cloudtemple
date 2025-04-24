package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual machine from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSVirtualMachineRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the virtual machine to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the virtual machine to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager to filter virtual machines by. Required when searching by `name`.",
			},

			// Out
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the virtual machine in the Open IaaS system.",
			},
			"power_state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current power state of the virtual machine (e.g., Running, Halted, Paused, etc...).",
			},
			"secure_boot": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether secure boot is enabled for the virtual machine.",
			},
			"boot_firmware": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The boot firmware type used by the virtual machine (e.g., BIOS, EFI).",
			},
			"auto_power_on": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the virtual machine is configured to automatically power on when the host starts.",
			},
			"dvd_drive": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the virtual machine's DVD drive.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the DVD drive.",
						},
						"attached": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the DVD drive is attached to the virtual machine.",
						},
					},
				},
			},
			"tools": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the virtualization tools installed in the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"detected": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether virtualization tools are detected in the virtual machine.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the virtualization tools installed in the virtual machine.",
						},
					},
				},
			},
			"boot_order": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The boot order of the virtual machine, listing devices in the order they will be tried during boot.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"operating_system_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the operating system installed on the virtual machine.",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of virtual CPUs allocated to the virtual machine.",
			},
			"num_cores_per_socket": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of cores per CPU socket in the virtual machine.",
			},
			"memory": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The amount of memory allocated to the virtual machine in Bytes.",
			},
			"addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The network addresses assigned to the virtual machine.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv6": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IPv6 address assigned to the virtual machine.",
						},
						"ipv4": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The IPv4 address assigned to the virtual machine.",
						},
					},
				},
			},
			"pool_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the resource pool the virtual machine belongs to.",
			},
			"host_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the host the virtual machine is running on.",
			},
		},
	}
}

// computeOpenIaaSVirtualMachineRead lit une machine virtuelle OpenIaaS et la mappe dans le state Terraform
func computeOpenIaaSVirtualMachineRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var vm *client.OpenIaaSVirtualMachine
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		virtualMachines, err := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &client.OpenIaaSVirtualMachineFilter{
			MachineManagerID: d.Get("machine_manager_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual machine named %q: %s", name, err))
		}
		for _, v := range virtualMachines {
			if v.Name == name {
				vm = v
				break
			}
		}
		if vm == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual machine named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			vm, err = c.Compute().OpenIaaS().VirtualMachine().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if vm == nil {
				return diag.FromErr(fmt.Errorf("failed to find virtual machine with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(vm.ID)

	// Mapper les données en utilisant la fonction helper
	vmData := helpers.FlattenOpenIaaSVirtualMachine(vm)

	// Définir les données dans le state
	for k, v := range vmData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
