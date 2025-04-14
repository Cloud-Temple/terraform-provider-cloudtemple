package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceResourcePools() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all resource pools from a vCenter infrastructure.",

		ReadContext: computeResourcePoolsRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"host_cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"resource_pools": {
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
						"machine_manager": {
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
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"parent": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"metrics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_usage": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"reservation_used": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"memory": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_usage": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"reservation_used": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"ballooned_memory": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
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

// computeResourcePoolsRead lit les pools de ressources et les mappe dans le state Terraform
func computeResourcePoolsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les pools de ressources
	resourcePools, err := c.Compute().ResourcePool().List(ctx, &client.ResourcePoolFilter{
		MachineManagerID: d.Get("machine_manager_id").(string),
		DatacenterID:     d.Get("datacenter_id").(string),
		HostClusterID:    d.Get("host_cluster_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("resource_pools")

	// Mapper manuellement les données en utilisant la fonction helper
	tfResourcePools := make([]map[string]interface{}, len(resourcePools))
	for i, pool := range resourcePools {
		tfResourcePools[i] = helpers.FlattenResourcePool(pool)
	}

	// Définir les données dans le state
	if err := d.Set("resource_pools", tfResourcePools); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
