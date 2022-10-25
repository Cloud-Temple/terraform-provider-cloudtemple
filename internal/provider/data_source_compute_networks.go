package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceNetworksRead,

		Schema: map[string]*schema.Schema{
			"networks": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"host_names": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	networks, err := client.Compute().Network().List(ctx, "", "", "", "", "", "", "", "", true)
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(networks))
	for i, network := range networks {
		res[i] = map[string]interface{}{
			"id":                      network.ID,
			"name":                    network.Name,
			"moref":                   network.Moref,
			"machine_manager_id":      network.MachineManagerId,
			"virtual_machines_number": network.VirtualMachinesNumber,
			"host_number":             network.HostNumber,
			"host_names":              network.HostNames,
		}
	}

	sw := newStateWriter(d, "networks")
	sw.set("networks", res)

	return sw.diags
}
