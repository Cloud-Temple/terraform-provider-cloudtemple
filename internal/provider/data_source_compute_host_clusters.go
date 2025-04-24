package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceHostClusters() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of host clusters.",

		ReadContext: computeHostClustersRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter host clusters by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter host clusters by machine manager ID.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter host clusters by datacenter ID.",
			},
			"datastore_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter host clusters by datastore ID.",
			},
			"datastore_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter host clusters by datastore cluster ID.",
			},

			// Out
			"host_clusters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of host clusters matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the host cluster.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the host cluster.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the host cluster.",
						},
						"hosts": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of hosts that are part of this host cluster.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The ID of the host.",
									},
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the host.",
									},
								},
							},
						},
						"metrics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Resource metrics for the host cluster.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"total_cpu": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total CPU capacity of the host cluster in MHz.",
									},
									"total_memory": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total memory capacity of the host cluster in bytes.",
									},
									"total_storage": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total storage capacity of the host cluster in bytes.",
									},
									"cpu_used": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of CPU currently used in the host cluster in MHz.",
									},
									"memory_used": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of memory currently used in the host cluster in bytes.",
									},
									"storage_used": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of storage currently used in the host cluster in bytes.",
									},
								},
							},
						},
						"virtual_machines_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines running on this host cluster.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this host cluster belongs to.",
						},
						"datacenter_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datacenter this host cluster belongs to.",
						},
					},
				},
			},
		},
	}
}

// computeHostClustersRead lit les clusters d'hôtes et les mappe dans le state Terraform
func computeHostClustersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les clusters d'hôtes
	hostClusters, err := c.Compute().HostCluster().List(ctx, &client.HostClusterFilter{
		Name:               d.Get("name").(string),
		MachineManagerId:   d.Get("machine_manager_id").(string),
		DatacenterId:       d.Get("datacenter_id").(string),
		DatastoreId:        d.Get("datastore_id").(string),
		DatastoreClusterId: d.Get("datastore_cluster_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("host_clusters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfHostClusters := make([]map[string]interface{}, len(hostClusters))
	for i, hostCluster := range hostClusters {
		tfHostClusters[i] = helpers.FlattenHostCluster(hostCluster)
	}

	// Définir les données dans le state
	if err := d.Set("host_clusters", tfHostClusters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
