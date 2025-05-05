package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceResourcePools() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all resource pools from a vCenter infrastructure.",

		ReadContext: computeResourcePoolsRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter resource pools by the ID of the machine manager they belong to.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter resource pools by the ID of the datacenter they belong to.",
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter resource pools by the ID of the host cluster they belong to.",
			},

			// Out
			"resource_pools": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of resource pools matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the resource pool.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the resource pool.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this resource pool belongs to.",
						},
						"machine_manager": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the machine manager this resource pool belongs to.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the machine manager.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the machine manager.",
									},
								},
							},
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the resource pool in the hypervisor.",
						},
						"parent": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the parent of this resource pool.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the parent object.",
									},
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the parent object (e.g., ResourcePool, HostCluster).",
									},
								},
							},
						},
						"metrics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Resource usage metrics for this resource pool.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cpu": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "CPU usage metrics for this resource pool.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_usage": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The maximum CPU usage in MHz.",
												},
												"reservation_used": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The amount of reserved CPU in MHz that is currently being used.",
												},
											},
										},
									},
									"memory": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Memory usage metrics for this resource pool.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_usage": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The maximum memory usage in MiB.",
												},
												"reservation_used": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The amount of reserved memory in MiB that is currently being used.",
												},
												"ballooned_memory": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The amount of memory in MiB that has been reclaimed by the balloon driver.",
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
