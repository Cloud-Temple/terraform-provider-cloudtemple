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
		Description: "",

		ReadContext: computeHostClusterRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"datastore_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"datastore_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
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
