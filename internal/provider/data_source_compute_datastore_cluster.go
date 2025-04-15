package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastoreCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific datastore cluster.",

		ReadContext: computeDatastoreClusterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datastore cluster to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the datastore cluster to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				Description:   "The ID of the machine manager to filter datastore clusters by. Only used when searching by name.",
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name", "datacenter_id"},
				Description:   "The ID of the datacenter to filter datastore clusters by. Required when searching by name.",
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				Description:   "The ID of the host to filter datastore clusters by. Only used when searching by name.",
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				Description:   "The ID of the host cluster to filter datastore clusters by. Only used when searching by name.",
			},

			// Out
			"moref": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The managed object reference ID of the datastore cluster.",
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
	}
}

// computeDatastoreClusterRead lit un cluster de datastore et le mappe dans le state Terraform
func computeDatastoreClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var datastoreCluster *client.DatastoreCluster
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		clusters, err := c.Compute().DatastoreCluster().List(ctx, &client.DatastoreClusterFilter{
			Name:             name,
			MachineManagerId: d.Get("machine_manager_id").(string),
			DatacenterId:     d.Get("datacenter_id").(string),
			HostId:           d.Get("host_id").(string),
			HostClusterId:    d.Get("host_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find datastore cluster named %q: %s", name, err))
		}
		for _, c := range clusters {
			if c.Name == name {
				datastoreCluster = c
				break
			}
		}
		if datastoreCluster == nil {
			return diag.FromErr(fmt.Errorf("failed to find datastore cluster named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		datastoreCluster, err = c.Compute().DatastoreCluster().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if datastoreCluster == nil {
			return diag.FromErr(fmt.Errorf("failed to find datastore cluster with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(datastoreCluster.ID)

	// Mapper les données en utilisant la fonction helper
	datastoreClusterData := helpers.FlattenDatastoreCluster(datastoreCluster)

	// Définir les données dans le state
	for k, v := range datastoreClusterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
