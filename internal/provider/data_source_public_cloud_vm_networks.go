package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourcePublicCloudVMNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve the full Public Cloud VM Instances network catalogue of the tenant. The documented server-side filters are not wired through the broker, so this datasource returns the complete catalogue and filtering must be done in HCL.",

		ReadContext: publicCloudVMNetworksRead,

		Schema: map[string]*schema.Schema{
			// Out
			"networks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of networks in the catalogue.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the network.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the network.",
						},
					},
				},
			},
		},
	}
}

func publicCloudVMNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	networks, err := c.PublicCloudVM().Network().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_networks")

	tfNetworks := make([]map[string]interface{}, len(networks))
	for i, network := range networks {
		tfNetworks[i] = helpers.FlattenPublicCloudVMNetwork(network)
	}

	if err := d.Set("networks", tfNetworks); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
