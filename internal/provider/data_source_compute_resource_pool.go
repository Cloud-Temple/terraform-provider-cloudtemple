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

func dataSourceResourcePool() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific resource pool from a vCenter infrastructure.",

		ReadContext: computeResourcePoolRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},

			// Out
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
	}
}

// computeResourcePoolRead lit un pool de ressources et le mappe dans le state Terraform
func computeResourcePoolRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var pool *client.ResourcePool
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		resourcePools, err := c.Compute().ResourcePool().List(ctx, &client.ResourcePoolFilter{
			MachineManagerID: d.Get("machine_manager_id").(string),
			DatacenterID:     d.Get("datacenter_id").(string),
			HostClusterID:    d.Get("host_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find resource pool named %q: %s", name, err))
		}
		for _, p := range resourcePools {
			if p.Name == name {
				pool = p
				break
			}
		}
		if pool == nil {
			return diag.FromErr(fmt.Errorf("failed to find resource pool named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			pool, err = c.Compute().ResourcePool().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if pool == nil {
				return diag.FromErr(fmt.Errorf("failed to find resource pool with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(pool.ID)

	// Mapper les données en utilisant la fonction helper
	poolData := helpers.FlattenResourcePool(pool)

	// Définir les données dans le state
	for k, v := range poolData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
