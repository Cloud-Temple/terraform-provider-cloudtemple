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
				Description:  "Filter networks by machine manager ID.",
			},
			"pool_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter networks by pool ID.",
			},

			// Out
			"networks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of networks matching the filter criteria.",

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
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the network in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this network belongs to.",
						},
						"pool_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the pool this network belongs to.",
						},
						"maximum_transmission_unit": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The Maximum Transmission Unit (MTU) size in bytes for this network.",
						},
						"network_adapters": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of network adapter IDs connected to this network.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"network_block_device": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this network supports network block devices.",
						},
						"insecure_network_block_device": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether this network allows insecure network block devices.",
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
