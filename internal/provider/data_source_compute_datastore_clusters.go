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
		Description: "Used to retrieve a list of datastore clusters.",

		ReadContext: computeDatastoreClustersRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastore clusters by name.",
			},
			"machine_manager_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastore clusters by machine manager ID.",
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastore clusters by datacenter ID.",
			},
			"host_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastore clusters by host ID.",
			},
			"host_cluster_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastore clusters by host cluster ID.",
			},
			// Out
			"datastore_clusters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of datastore clusters matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the datastore cluster.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the datastore cluster.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the datastore cluster.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this datastore cluster belongs to.",
						},
						"datacenter_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datacenter this datastore cluster belongs to.",
						},
						"datastores": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of datastore IDs that are part of this datastore cluster.",

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"metrics": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Metrics and configuration information for the datastore cluster.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free_capacity": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The free capacity of the datastore cluster in bytes.",
									},
									"max_capacity": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The maximum capacity of the datastore cluster in bytes.",
									},
									"enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether Storage DRS is enabled for this datastore cluster.",
									},
									"default_vm_behavior": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The default Storage DRS behavior for virtual machines.",
									},
									"load_balance_interval": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Specify the interval that storage DRS runs to load balance among datastores within a storage pod (in minutes)",
									},
									"space_threshold_mode": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The space utilization threshold mode for Storage DRS.",
									},
									"space_utilization_threshold": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Storage DRS makes storage migration recommendations if space utilization on one (or more) of the datastores is higher than the specified threshold. The valid values are in the range of 50 (i.e., 50%) to 100 (i.e., 100%). If not specified, the default value is 80%.",
									},
									"min_space_utilization_difference": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Storage DRS considers making storage migration recommendations if the difference in space utilization between the source and destination datastores is higher than the specified threshold. The valid values are in the range of 1 (i.e., 1%) to 50 (i.e., 50%). If not specified, the default value is 5%.",
									},
									"reservable_percent_threshold": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Storage DRS makes storage migration recommendations if total IOPs reservation of all VMs running on a datastore is higher than the specified threshold. Storage DRS recommends migration out of all such datastores, if more than one datastore exceed their reserved IOPs threshold. The actual Iops used to determine threshold are computed from Storage DRS estimation of IOPs capacity of a datastore. The absolute value may change over time, according to storage response to workloads. The valid values are in the range of 30 (i.e., 30%) to 100 (i.e., 100%). If not specified, the default value is 60%.",
									},
									"reservable_threshold_mode": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Determines which reservation threshold specification to use. If unspecified, the mode is assumed automatic by default. Storage DRS uses percentage value in that case. If mode is specified, but corresponding reservationThreshold value is absent, option specific defaults are used.",
									},
									"io_latency_threshold": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Storage DRS makes storage migration recommendations if I/O latency on one (or more) of the datastores is higher than the specified threshold (millisecond). The valid values are in the range of 5 to 100. If not specified, the default value is 15.",
									},
									"io_load_imbalance_threshold": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "Storage DRS makes storage migration recommendations if I/O load imbalance level is higher than the specified threshold (number). The valid values are in the range of 1 to 100. If not specified, the default value is 5.",
									},
									"io_load_balance_enabled": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Flag indicating whether or not storage DRS takes into account storage I/O workload when making load balancing and initial placement recommendations.",
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
