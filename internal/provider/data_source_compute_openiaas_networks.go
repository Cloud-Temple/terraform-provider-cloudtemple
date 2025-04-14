package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all networks from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSNetworksRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"pool_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
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
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"maximum_transmission_unit": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"network_adapters": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"network_block_device": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"insecure_network_block_device": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSNetworksRead lit les réseaux OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les réseaux OpenIaaS
	networks, err := c.Compute().OpenIaaS().Network().List(ctx, &client.OpenIaaSNetworkFilter{
		MachineManagerID: d.Get("machine_manager_id").(string),
		PoolID:           d.Get("pool_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_networks")

	// Mapper manuellement les données en utilisant la fonction helper
	tfNetworks := make([]map[string]interface{}, len(networks))
	for i, network := range networks {
		tfNetworks[i] = helpers.FlattenOpenIaaSNetwork(network)
	}

	// Définir les données dans le state
	if err := d.Set("networks", tfNetworks); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
