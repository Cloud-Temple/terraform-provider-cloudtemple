package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasMachineManagers() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all machine managers from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSMachineManagersRead,

		Schema: map[string]*schema.Schema{
			// Out
			"availability_zones": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all machine managers in the Open IaaS infrastructure.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the machine manager.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the machine manager.",
						},
						"os_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The operating system version of the machine manager.",
						},
						"os_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The operating system name of the machine manager.",
						},
						"xoa_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The XOA version of the machine manager.",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSMachineManagersRead lit les gestionnaires de machines OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSMachineManagersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les gestionnaires de machines OpenIaaS
	machineManagers, err := c.Compute().OpenIaaS().MachineManager().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_availability_zones")

	// Mapper manuellement les données en utilisant la fonction helper
	tfMachineManagers := make([]map[string]interface{}, len(machineManagers))
	for i, machineManager := range machineManagers {
		tfMachineManagers[i] = helpers.FlattenOpenIaaSMachineManager(machineManager)
	}

	// Définir les données dans le state
	if err := d.Set("availability_zones", tfMachineManagers); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
