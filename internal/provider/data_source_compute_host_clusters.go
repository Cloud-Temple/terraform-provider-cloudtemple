package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeHostClustersRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
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
			"datastore_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datastore_cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"host_clusters": {
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
						"hosts": {
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
									"total_cpu": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total_memory": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total_storage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cpu_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"memory_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"storage_used": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"virtual_machines_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datacenter_id": {
							Type:     schema.TypeString,
							Computed: true,
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
