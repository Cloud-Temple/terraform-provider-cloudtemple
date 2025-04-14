package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDatacenters() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual datacenters from a vCenter infrastructure.",

		ReadContext: computeVirtualDatacentersRead,

		Schema: map[string]*schema.Schema{
			// Out
			"virtual_datacenters": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vcenter": {
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
						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// computeVirtualDatacentersRead lit les datacenters virtuels et les mappe dans le state Terraform
func computeVirtualDatacentersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les datacenters virtuels
	datacenters, err := c.Compute().VirtualDatacenter().List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("virtual_datacenters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfDatacenters := make([]map[string]interface{}, len(datacenters))
	for i, datacenter := range datacenters {
		tfDatacenters[i] = helpers.FlattenVirtualDatacenter(datacenter)
	}

	// Définir les données dans le state
	if err := d.Set("virtual_datacenters", tfDatacenters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
