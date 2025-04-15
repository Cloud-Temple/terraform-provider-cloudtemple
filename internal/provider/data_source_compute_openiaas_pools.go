package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasPools() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all pools from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSPoolsRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter pools by machine manager ID.",
			},

			// Out
			"pools": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of pools matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the pool.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the pool.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the pool in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this pool belongs to.",
						},
						"high_availability_enabled": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether high availability is enabled for this pool.",
						},
						"hosts": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of host IDs in this pool.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"cpu": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "CPU information for the pool.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cores": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of CPU cores in the pool.",
									},
									"sockets": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of CPU sockets in the pool.",
									},
								},
							},
						},
						"type": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the pool type.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The key identifier of the pool type.",
									},
									"description": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The description of the pool type.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSPoolsRead lit les pools OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSPoolsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les pools OpenIaaS
	pools, err := c.Compute().OpenIaaS().Pool().List(ctx, &client.OpenIaasPoolFilter{
		MachineManagerId: d.Get("machine_manager_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_pools")

	// Mapper manuellement les données en utilisant la fonction helper
	tfPools := make([]map[string]interface{}, len(pools))
	for i, pool := range pools {
		tfPools[i] = helpers.FlattenOpenIaaSPool(pool)
	}

	// Définir les données dans le state
	if err := d.Set("pools", tfPools); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
