package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualSwitchs() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual switches from a vCenter infrastructure.",

		ReadContext: dataSourceVirtualSwitchsRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter virtual switches by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter virtual switches by machine manager ID.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter virtual switches by datacenter ID.",
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter virtual switches by host cluster ID.",
			},

			// Out
			"virtual_switchs": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual switches matching the specified filters.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual switch.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual switch.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the virtual switch in the hypervisor.",
						},
						"folder_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the folder where the virtual switch is located.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager (vCenter) where the virtual switch is located.",
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
