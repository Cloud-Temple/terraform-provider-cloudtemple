package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastoreClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeDatastoreClustersRead,

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
			"host_id": {
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
			"datastore_clusters": {
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
						"datacenter_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastores": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metrics": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free_capacity": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"max_capacity": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"enabled": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"default_vm_behavior": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"load_balance_interval": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"space_threshold_mode": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"space_utilization_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"min_space_utilization_difference": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservable_percent_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"reservable_threshold_mode": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"io_latency_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"io_load_imbalance_threshold": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"io_load_balance_enabled": {
										Type:     schema.TypeBool,
										Computed: true,
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

// computeDatastoreClustersRead lit les clusters de datastores et les mappe dans le state Terraform
func computeDatastoreClustersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les clusters de datastores
	datastoreClusters, err := c.Compute().DatastoreCluster().List(ctx, &client.DatastoreClusterFilter{
		Name:             d.Get("name").(string),
		MachineManagerId: d.Get("machine_manager_id").(string),
		DatacenterId:     d.Get("datacenter_id").(string),
		HostId:           d.Get("host_id").(string),
		HostClusterId:    d.Get("host_cluster_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("datastore_clusters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfDatastoreClusters := make([]map[string]interface{}, len(datastoreClusters))
	for i, datastoreCluster := range datastoreClusters {
		tfDatastoreClusters[i] = helpers.FlattenDatastoreCluster(datastoreCluster)
	}

	// Définir les données dans le state
	if err := d.Set("datastore_clusters", tfDatastoreClusters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
