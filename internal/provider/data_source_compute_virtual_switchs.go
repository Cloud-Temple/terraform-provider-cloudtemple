package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualSwitchs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual switches from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualSwitchsRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"host_cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			// Out
			"virtual_switchs": {
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
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"folder_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// dataSourceVirtualSwitchsRead lit les commutateurs virtuels et les mappe dans le state Terraform
func dataSourceVirtualSwitchsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les commutateurs virtuels
	virtualSwitches, err := c.Compute().VirtualSwitch().List(ctx, &client.VirtualSwitchFilter{
		Name:             d.Get("name").(string),
		MachineManagerId: d.Get("machine_manager_id").(string),
		DatacenterId:     d.Get("datacenter_id").(string),
		HostClusterId:    d.Get("host_cluster_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("virtual_switchs")

	// Mapper manuellement les données en utilisant la fonction helper
	tfVirtualSwitches := make([]map[string]interface{}, len(virtualSwitches))
	for i, vs := range virtualSwitches {
		tfVirtualSwitches[i] = helpers.FlattenVirtualSwitch(vs)
	}

	// Définir les données dans le state
	if err := d.Set("virtual_switchs", tfVirtualSwitches); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
