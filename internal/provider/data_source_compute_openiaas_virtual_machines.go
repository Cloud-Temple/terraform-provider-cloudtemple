package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasVirtualMachines() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual machines from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSVirtualMachinesRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"virtual_machines": {
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
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
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
						"host_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSVirtualMachinesRead lit les machines virtuelles OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSVirtualMachinesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les machines virtuelles OpenIaaS
	vms, err := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &client.OpenIaaSVirtualMachineFilter{
		MachineManagerID: d.Get("machine_manager_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_virtual_machines")

	// Mapper manuellement les données en utilisant la fonction helper
	tfVMs := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		tfVMs[i] = helpers.FlattenOpenIaaSVirtualMachine(vm)
		tfVMs[i]["id"] = vm.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_machines", tfVMs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
