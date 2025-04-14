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
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secure_boot": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"boot_firmware": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_power_on": {
				Type:     schema.TypeBool,
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
			"boot_order": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"operating_system_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"memory": {
				Type:     schema.TypeInt,
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
			"pool_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"host_id": {
				Type:     schema.TypeString,
				Computed: true,
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
