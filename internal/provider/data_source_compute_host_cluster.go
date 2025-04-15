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

func dataSourceHostCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific host cluster.",

		ReadContext: computeHostClusterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host cluster to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the host cluster to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the machine manager to filter host clusters by. Only used when searching by name.",
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the datacenter to filter host clusters by. Only used when searching by name.",
			},
			"datastore_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the datastore to filter host clusters by. Only used when searching by name.",
			},
			"datastore_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the datastore cluster to filter host clusters by. Only used when searching by name.",
			},

			// Out
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
							Description: "The total memory capacity of the host cluster in MiB.",
						},
						"total_storage": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The total storage capacity of the host cluster in MiB.",
						},
						"cpu_used": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of CPU currently used in the host cluster in MHz.",
						},
						"memory_used": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of memory currently used in the host cluster in MiB.",
						},
						"storage_used": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The amount of storage currently used in the host cluster in MiB.",
						},
					},
				},
			},
			"virtual_machines_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of virtual machines running on this host cluster.",
			},
		},
	}
}

// computeHostClusterRead lit un cluster d'hôtes et le mappe dans le state Terraform
func computeHostClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var hostCluster *client.HostCluster
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		hostClusters, err := c.Compute().HostCluster().List(ctx, &client.HostClusterFilter{
			Name:               name,
			MachineManagerId:   d.Get("machine_manager_id").(string),
			DatacenterId:       d.Get("datacenter_id").(string),
			DatastoreId:        d.Get("datastore_id").(string),
			DatastoreClusterId: d.Get("datastore_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find host cluster named %q: %s", name, err))
		}
		for _, hc := range hostClusters {
			if hc.Name == name {
				hostCluster = hc
				break
			}
		}
		if hostCluster == nil {
			return diag.FromErr(fmt.Errorf("failed to find host cluster named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		hostCluster, err = c.Compute().HostCluster().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if hostCluster == nil {
			return diag.FromErr(fmt.Errorf("failed to find host cluster with id %q", id))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(hostCluster.ID)

	// Mapper les données en utilisant la fonction helper
	hostClusterData := helpers.FlattenHostCluster(hostCluster)

	// Définir les données dans le state
	for k, v := range hostClusterData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
